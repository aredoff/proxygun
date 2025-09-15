package proxygun

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/aredoff/proxygun/internal/parser"
	"github.com/aredoff/proxygun/internal/pool"
	"github.com/aredoff/proxygun/internal/proxy"
	"github.com/aredoff/proxygun/internal/validator"
	netproxy "golang.org/x/net/proxy"
)

type Config struct {
	PoolSize          int
	MaxRetries        int
	RefreshInterval   time.Duration
	ValidationWorkers int
	BadProxyMaxAge    time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		PoolSize:          50,
		MaxRetries:        3,
		RefreshInterval:   10 * time.Second,
		ValidationWorkers: 30,
		BadProxyMaxAge:    24 * time.Hour,
	}
}

type Client struct {
	config    *Config
	pool      *pool.Pool
	parser    *parser.MultiParser
	validator *validator.Validator
	mu        sync.RWMutex
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	client := &Client{
		config:    config,
		pool:      pool.NewPool(config.PoolSize),
		parser:    parser.NewMultiParser(),
		validator: validator.NewValidator(),
		stopCh:    make(chan struct{}),
	}

	client.startBackgroundTasks()
	return client
}

func (c *Client) startBackgroundTasks() {
	c.wg.Add(2)

	go c.proxyRefreshWorker()
	go c.badProxyCleanupWorker()
}

func (c *Client) proxyRefreshWorker() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.RefreshInterval)
	defer ticker.Stop()

	c.refreshProxies()

	for {
		select {
		case <-ticker.C:
			needed := c.pool.NeedsProxies()
			if needed > 0 || c.pool.FreeSize() == 0 {
				c.refreshProxies()
			}
		case <-c.stopCh:
			return
		}
	}
}

func (c *Client) badProxyCleanupWorker() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.pool.CleanOldBadProxies(c.config.BadProxyMaxAge)
			c.pool.CheckBadProxies()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Client) refreshProxies() {
	proxies, errs := c.parser.ParseAll()
	providerName := c.parser.GetCurrentProviderName()

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
		c.validator.ValidateProxiesConcurrentStream(proxies, c.config.ValidationWorkers, validChan)
	}()

	added := 0
	for validProxy := range validChan {
		if c.pool.Add(validProxy) {
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

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
		proxyWithStats := c.pool.GetNext()
		if proxyWithStats == nil {
			return nil, errors.New("no proxies available")
		}

		resp, err := c.doWithProxy(req, proxyWithStats)
		if err != nil {
			proxyWithStats.RecordFailure()
			lastErr = err
			continue
		}

		proxyWithStats.RecordSuccess()
		return resp, nil
	}

	return nil, fmt.Errorf("all proxy attempts failed, last error: %w", lastErr)
}

func (c *Client) doWithProxy(req *http.Request, proxyWithStats *proxy.ProxyWithStats) (*http.Response, error) {
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

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return client.Do(req)
}

func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	return c.Do(req)
}

func (c *Client) Post(url, contentType string, body interface{}) (*http.Response, error) {
	// Implementation similar to http.Client.Post
	// This would need proper body handling
	return nil, errors.New("not implemented yet")
}

func (c *Client) Stats() map[string]interface{} {
	return map[string]interface{}{
		"pool_size":      c.pool.Size(),
		"free_pool_size": c.pool.FreeSize(),
		"needs_proxies":  c.pool.NeedsProxies(),
	}
}

func (c *Client) Close() error {
	close(c.stopCh)
	c.wg.Wait()
	return nil
}
