package validator

import (
	"time"

	"github.com/aredoff/proxygun/internal/proxy"
)

type Validator struct {
	timeout      time.Duration
	testURL      string
	testHeaders  map[string]string
	maxRetries   int
	tcpTimeout   time.Duration
	skipTCPCheck bool
}

func NewValidator() *Validator {
	return &Validator{
		timeout: 5 * time.Second,
		testURL: "https://www.ripe.net",
		testHeaders: map[string]string{
			"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Language":           "en-US,en;q=0.9",
			"Accept-Encoding":           "gzip, deflate, br",
			"Connection":                "keep-alive",
			"Upgrade-Insecure-Requests": "1",
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "none",
		},
		maxRetries:   2,
		tcpTimeout:   2 * time.Second,
		skipTCPCheck: false,
	}
}

// NewValidatorWithOptions creates validator with custom options
func NewValidatorWithOptions(timeout, tcpTimeout time.Duration, skipTCPCheck bool) *Validator {
	return &Validator{
		timeout:      timeout,
		testURL:      "https://www.ripe.net",
		maxRetries:   2,
		tcpTimeout:   tcpTimeout,
		skipTCPCheck: skipTCPCheck,
	}
}

func (v *Validator) ValidateProxy(p *proxy.Proxy) bool {
	// Quick TCP connectivity check first
	if !v.skipTCPCheck && !v.checkTCPConnectivity(p) {
		return false
	}

	for i := 0; i < v.maxRetries; i++ {
		if v.testProxy(p) {
			return true
		}
	}
	return false
}

// ValidateAndDetectType validates proxy and automatically detects its type
func (v *Validator) ValidateAndDetectType(p *proxy.Proxy) (*proxy.Proxy, bool) {
	// Quick TCP connectivity check first
	if !v.skipTCPCheck && !v.checkTCPConnectivity(p) {
		return nil, false
	}

	// Try HTTP first (most common)
	if v.tryProxyType(p, proxy.HTTP) {
		return &proxy.Proxy{
			Host: p.Host,
			Port: p.Port,
			Type: proxy.HTTP,
		}, true
	}

	// Try SOCKS5
	if v.tryProxyType(p, proxy.SOCKS5) {
		return &proxy.Proxy{
			Host: p.Host,
			Port: p.Port,
			Type: proxy.SOCKS5,
		}, true
	}

	// Try SOCKS4 (least common, try last)
	if v.tryProxyType(p, proxy.SOCKS4) {
		return &proxy.Proxy{
			Host: p.Host,
			Port: p.Port,
			Type: proxy.SOCKS4,
		}, true
	}

	return nil, false
}

func (v *Validator) tryProxyType(p *proxy.Proxy, proxyType proxy.Type) bool {
	testProxy := &proxy.Proxy{
		Host: p.Host,
		Port: p.Port,
		Type: proxyType,
	}

	// Skip TCP check here since it's already done in ValidateAndDetectType
	for i := 0; i < v.maxRetries; i++ {
		if v.testProxy(testProxy) {
			return true
		}
	}
	return false
}

func (v *Validator) testProxy(p *proxy.Proxy) bool {
	switch p.Type {
	case proxy.HTTP:
		return v.testHTTPProxy(p)
	case proxy.SOCKS4, proxy.SOCKS5:
		return v.testSOCKSProxy(p)
	default:
		return false
	}
}
