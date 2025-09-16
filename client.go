package proxygun

import (
	"net/http"
	"time"

	"github.com/aredoff/proxygun/internal/parser"
	"github.com/aredoff/proxygun/internal/pool"
	"github.com/aredoff/proxygun/internal/proxy"
	"github.com/aredoff/proxygun/internal/validator"
)

type ProxyRoundTripper struct {
	config    *Config
	pool      *pool.Pool
	parser    *parser.MultiParser
	validator *validator.Validator
	stopCh    chan struct{}
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

	go rt.proxyRefreshWorker()
	return rt
}

func (rt *ProxyRoundTripper) proxyRefreshWorker() {

	ticker := time.NewTicker(rt.config.RefreshInterval)
	defer ticker.Stop()

	rt.refreshProxies()

	for {
		select {
		case <-ticker.C:
			// Move proxies from free pool to main pool if needed
			beforeSize := rt.pool.Size()
			rt.pool.FillFromFree()
			afterSize := rt.pool.Size()

			if afterSize > beforeSize {
				rt.config.Logger.Info().Msgf("Moved %d proxies from free pool to main pool (%d -> %d)",
					afterSize-beforeSize, beforeSize, afterSize)
			}

			needed := rt.pool.NeedsProxies()
			if needed > 0 {
				rt.refreshProxies()
			}
		case <-rt.stopCh:
			return
		}
	}
}

func (rt *ProxyRoundTripper) refreshProxies() {
	proxies, errs := rt.parser.Parse()
	providerName := rt.parser.GetCurrentProviderName()

	if len(errs) > 0 {
		for _, err := range errs {
			rt.config.Logger.Error().Msgf("Parser error from %s: %v", providerName, err)
		}
	}

	if len(proxies) == 0 {
		rt.config.Logger.Info().Msgf("No proxies found from %s", providerName)
		return
	}
	rt.config.Logger.Info().Msgf("Found %d proxies from %s, starting validation...", len(proxies), providerName)

	// Start validation in background and add proxies as they get validated
	validChan := make(chan *proxy.Proxy, 100)
	go func() {
		defer close(validChan)
		ValidateProxiesConcurrentStream(rt.validator, proxies, rt.config.ValidationWorkers, validChan)
	}()

	added := 0
	for validProxy := range validChan {
		if rt.pool.Add(validProxy) {
			added++
		}
	}

	if added > 0 {
		rt.config.Logger.Info().Msgf("Added %d new proxies to pool from %s (validated %d from %d found)",
			added, providerName, added, len(proxies))
	} else {
		rt.config.Logger.Info().Msgf("No valid proxies found from %s (checked %d)", providerName, len(proxies))
	}
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
	return nil
}

// ProxyClient wraps http.Client with proxy functionality
type ProxyClient struct {
	*http.Client
	rt *ProxyRoundTripper
}

// NewProxyClient creates a new HTTP client with proxy rotation
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
func (c *ProxyClient) Stats() map[string]interface{} {
	return c.rt.Stats()
}

// Close stops background workers and cleans up resources
func (c *ProxyClient) Close() error {
	return c.rt.Close()
}
