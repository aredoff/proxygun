package proxygun

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/aredoff/proxygun/internal/proxy"
	"h12.io/socks"
)

// RoundTrip implements the http.RoundTripper interface
func (rt *ProxyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastErr error
	var proxyAttempts int

	// Try with proxies first
	for attempt := 0; attempt < rt.config.MaxRetries; attempt++ {
		proxyWithStats := rt.pool.Next()
		if proxyWithStats == nil {
			break // No proxies available
		}

		if proxyWithStats.Stats.IsBad(MinimalRequestsToCheckBad) {
			rt.pool.MoveToBad(proxyWithStats.Proxy)
			rt.config.Logger.Info().Msgf("Proxy %s is bad, moving to bad pool", proxyWithStats.Proxy.String())
			attempt--
			continue
		}

		proxyAttempts++
		resp, err := rt.roundTripWithProxy(req, proxyWithStats)
		if err != nil {
			proxyWithStats.RecordFailure()
			lastErr = err
			continue
		}

		if !slices.Contains(rt.config.GoodCodes, resp.StatusCode) {
			proxyWithStats.RecordFailure()
			lastErr = fmt.Errorf("status code: %d", resp.StatusCode)
			continue
		}

		proxyWithStats.RecordSuccess()
		return resp, nil
	}

	// If no proxies available or all proxies failed, use fallback transport
	if rt.config.FallbackTransport != nil {
		resp, err := rt.config.FallbackTransport.RoundTrip(req)
		if err != nil {
			if proxyAttempts > 0 {
				return nil, fmt.Errorf("all %d proxy attempts failed (last proxy error: %v), fallback transport also failed: %w", proxyAttempts, lastErr, err)
			}
			return nil, fmt.Errorf("no proxies available, fallback transport failed: %w", err)
		}
		return resp, nil
	}

	// No fallback transport configured
	if proxyAttempts > 0 {
		return nil, fmt.Errorf("all %d proxy attempts failed, last error: %w", proxyAttempts, lastErr)
	}
	return nil, errors.New("no proxies available and no fallback transport configured")
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
		proxyAddr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port))

		var proxyURI string
		switch p.Type {
		case proxy.SOCKS4:
			proxyURI = fmt.Sprintf("socks4://%s?timeout=30s", proxyAddr)
		case proxy.SOCKS5:
			proxyURI = fmt.Sprintf("socks5://%s?timeout=30s", proxyAddr)
		}

		dialSocksProxy := socks.Dial(proxyURI)
		if dialSocksProxy == nil {
			return nil, errors.New("failed to create SOCKS proxy dialer")
		}

		transport = &http.Transport{
			Dial:                dialSocksProxy,
			TLSHandshakeTimeout: 10 * time.Second,
		}
	default:
		return nil, errors.New("unsupported proxy type")
	}

	return transport.RoundTrip(req)
}
