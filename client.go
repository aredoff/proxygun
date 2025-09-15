package proxygun

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/aredoff/proxygun/internal/parser"
	"github.com/aredoff/proxygun/internal/pool"
	"github.com/aredoff/proxygun/internal/proxy"
	"github.com/aredoff/proxygun/internal/validator"
	"github.com/rs/zerolog"
	netproxy "golang.org/x/net/proxy"
)

type Config struct {
	PoolSize          int
	MaxRetries        int
	RefreshInterval   time.Duration
	ValidationWorkers int
	BadProxyMaxAge    time.Duration
	FallbackTransport http.RoundTripper
	Logger            zerolog.Logger
}

func DefaultConfig() *Config {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	return &Config{
		PoolSize:          50,
		MaxRetries:        3,
		RefreshInterval:   10 * time.Second,
		ValidationWorkers: 30,
		BadProxyMaxAge:    24 * time.Hour,
		FallbackTransport: http.DefaultTransport,
		Logger:            logger,
	}
}

type ProxyRoundTripper struct {
	config    *Config
	pool      *pool.Pool
	parser    *parser.MultiParser
	validator *validator.Validator
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewProxyRoundTripper(config *Config) *ProxyRoundTripper {
	if config == nil {
		config = DefaultConfig()
	}

	rt := &ProxyRoundTripper{
		config:    config,
		pool:      pool.NewPool(config.PoolSize),
		parser:    parser.NewMultiParser(),
		validator: validator.NewValidator(),
		stopCh:    make(chan struct{}),
	}

	rt.startBackgroundTasks()
	return rt
}

// NewClient creates an http.Client with ProxyRoundTripper
func NewClient(config *Config) *http.Client {
	rt := NewProxyRoundTripper(config)
	return &http.Client{
		Transport: rt,
		Timeout:   30 * time.Second,
	}
}

func (rt *ProxyRoundTripper) startBackgroundTasks() {
	rt.wg.Add(2)

	go rt.proxyRefreshWorker()
	go rt.badProxyCleanupWorker()
}

func (rt *ProxyRoundTripper) proxyRefreshWorker() {
	defer rt.wg.Done()

	ticker := time.NewTicker(rt.config.RefreshInterval)
	defer ticker.Stop()

	rt.refreshProxies()

	for {
		select {
		case <-ticker.C:
			needed := rt.pool.NeedsProxies()
			if needed > 0 || rt.pool.FreeSize() == 0 {
				rt.refreshProxies()
			}
		case <-rt.stopCh:
			return
		}
	}
}

func (rt *ProxyRoundTripper) badProxyCleanupWorker() {
	defer rt.wg.Done()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rt.pool.CleanOldBadProxies(rt.config.BadProxyMaxAge)
			rt.pool.CheckBadProxies()
		case <-rt.stopCh:
			return
		}
	}
}

func (rt *ProxyRoundTripper) refreshProxies() {
	proxies, errs := rt.parser.ParseAll()
	providerName := rt.parser.GetCurrentProviderName()

	if len(errs) > 0 {
		for _, err := range errs {
			log.Printf("Parser error from %s: %v", providerName, err)
		}
	}

	if len(proxies) == 0 {
		log.Printf("No proxies found from %s", providerName)
		return
	}

	log.Printf("Found %d proxies from %s, starting validation...", len(proxies), providerName)

	// Start validation in background and add proxies as they get validated
	validChan := make(chan *proxy.Proxy, 100)
	go func() {
		defer close(validChan)
		rt.validator.ValidateProxiesConcurrentStream(proxies, rt.config.ValidationWorkers, validChan)
	}()

	added := 0
	for validProxy := range validChan {
		if rt.pool.Add(validProxy) {
			added++
		}
	}

	if added > 0 {
		log.Printf("Added %d new proxies to pool from %s (validated %d from %d found)",
			added, providerName, added, len(proxies))
	} else {
		log.Printf("No valid proxies found from %s (checked %d)", providerName, len(proxies))
	}
}

// RoundTrip implements the http.RoundTripper interface
func (rt *ProxyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastErr error

	// Try with proxies first
	for attempt := 0; attempt < rt.config.MaxRetries; attempt++ {
		proxyWithStats := rt.pool.GetNext()
		if proxyWithStats == nil {
			break // No proxies available, try direct connection if allowed
		}

		resp, err := rt.roundTripWithProxy(req, proxyWithStats)
		if err != nil {
			proxyWithStats.RecordFailure()
			lastErr = err
			continue
		}

		proxyWithStats.RecordSuccess()
		return resp, nil
	}

	// If all proxies failed and fallback transport is configured, use it
	if rt.config.FallbackTransport != nil {
		resp, err := rt.config.FallbackTransport.RoundTrip(req)
		if err != nil {
			if lastErr != nil {
				return nil, fmt.Errorf("all proxy attempts failed (last proxy error: %v), fallback transport also failed: %w", lastErr, err)
			}
			return nil, fmt.Errorf("no proxies available, fallback transport failed: %w", err)
		}
		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all proxy attempts failed, last error: %w", lastErr)
	}
	return nil, errors.New("no proxies available")
}

func (rt *ProxyRoundTripper) roundTripWithProxy(req *http.Request, proxyWithStats *proxy.ProxyWithStats) (*http.Response, error) {
	p := proxyWithStats.Proxy

	var transport *http.Transport

	switch p.Type {
	case proxy.HTTP:
		transport = &http.Transport{
			Proxy: http.ProxyURL(p.URL()),
			DialContext: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		}
	case proxy.SOCKS4, proxy.SOCKS5:
		var dialer netproxy.Dialer
		var err error

		proxyAddr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port))

		if p.Type == proxy.SOCKS4 {
			return nil, errors.New("SOCKS4 not supported")
		} else {
			var auth *netproxy.Auth
			if p.Username != "" {
				auth = &netproxy.Auth{
					User:     p.Username,
					Password: p.Password,
				}
			}
			dialer, err = netproxy.SOCKS5("tcp", proxyAddr, auth, netproxy.Direct)
		}

		if err != nil {
			return nil, err
		}

		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
			TLSHandshakeTimeout: 10 * time.Second,
		}
	default:
		return nil, errors.New("unsupported proxy type")
	}

	return transport.RoundTrip(req)
}

// Stats returns current proxy pool statistics
func (rt *ProxyRoundTripper) Stats() map[string]interface{} {
	return map[string]interface{}{
		"pool_size":      rt.pool.Size(),
		"free_pool_size": rt.pool.FreeSize(),
		"needs_proxies":  rt.pool.NeedsProxies(),
	}
}

// Close stops background workers and cleans up resources
func (rt *ProxyRoundTripper) Close() error {
	close(rt.stopCh)
	rt.wg.Wait()
	return nil
}

// ProxyClient wraps http.Client with ProxyRoundTripper for convenience
type ProxyClient struct {
	*http.Client
	rt *ProxyRoundTripper
}

// NewProxyClient creates a ProxyClient with stats and close methods
func NewProxyClient(config *Config) *ProxyClient {
	rt := NewProxyRoundTripper(config)
	return &ProxyClient{
		Client: &http.Client{
			Transport: rt,
			Timeout:   30 * time.Second,
		},
		rt: rt,
	}
}

// Stats returns current proxy pool statistics
func (pc *ProxyClient) Stats() map[string]interface{} {
	return pc.rt.Stats()
}

// Close stops background workers and cleans up resources
func (pc *ProxyClient) Close() error {
	return pc.rt.Close()
}
