package validator

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/aredoff/proxygun/internal/proxy"
	netproxy "golang.org/x/net/proxy"
)

type Validator struct {
	timeout    time.Duration
	testURL    string
	maxRetries int
}

func NewValidator() *Validator {
	return &Validator{
		timeout:    10 * time.Second,
		testURL:    "https://www.google.com",
		maxRetries: 2,
	}
}

func (v *Validator) ValidateProxy(p *proxy.Proxy) bool {
	for i := 0; i < v.maxRetries; i++ {
		if v.testProxy(p) {
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

func (v *Validator) testHTTPProxy(p *proxy.Proxy) bool {
	proxyURL := p.URL()

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		DialContext: (&net.Dialer{
			Timeout: v.timeout,
		}).DialContext,
		TLSHandshakeTimeout: v.timeout,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   v.timeout,
	}

	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", v.testURL, nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (v *Validator) testSOCKSProxy(p *proxy.Proxy) bool {
	var dialer netproxy.Dialer
	var err error

	proxyAddr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port))

	if p.Type == proxy.SOCKS4 {
		return false
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
		return false
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		TLSHandshakeTimeout: v.timeout,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   v.timeout,
	}

	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", v.testURL, nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (v *Validator) ValidateProxies(proxies []*proxy.Proxy) []*proxy.Proxy {
	validProxies := make([]*proxy.Proxy, 0)

	for _, p := range proxies {
		if v.ValidateProxy(p) {
			validProxies = append(validProxies, p)
		}
	}

	return validProxies
}

func (v *Validator) ValidateProxiesConcurrent(proxies []*proxy.Proxy, workers int) []*proxy.Proxy {
	if workers <= 0 {
		workers = 10
	}

	jobs := make(chan *proxy.Proxy, len(proxies))
	results := make(chan *proxy.Proxy, len(proxies))

	for w := 0; w < workers; w++ {
		go func() {
			for p := range jobs {
				if v.ValidateProxy(p) {
					results <- p
				} else {
					results <- nil
				}
			}
		}()
	}

	for _, p := range proxies {
		jobs <- p
	}
	close(jobs)

	validProxies := make([]*proxy.Proxy, 0)
	processed := 0
	for i := 0; i < len(proxies); i++ {
		if result := <-results; result != nil {
			validProxies = append(validProxies, result)
		}
		processed++
	}

	return validProxies
}

func (v *Validator) ValidateProxiesConcurrentStream(proxies []*proxy.Proxy, workers int, validChan chan<- *proxy.Proxy) {
	if workers <= 0 {
		workers = 10
	}
	if workers > 50 {
		workers = 50
	}

	jobs := make(chan *proxy.Proxy, len(proxies))
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				if v.ValidateProxy(p) {
					validChan <- p
				}
			}
		}()
	}

	for _, p := range proxies {
		jobs <- p
	}
	close(jobs)

	wg.Wait()
}
