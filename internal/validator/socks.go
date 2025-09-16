package validator

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/aredoff/proxygun/internal/proxy"
	"h12.io/socks"
)

func (v *Validator) testSOCKSProxy(p *proxy.Proxy) bool {
	proxyAddr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port))

	var proxyURI string
	switch p.Type {
	case proxy.SOCKS4:
		proxyURI = fmt.Sprintf("socks4://%s?timeout=%s", proxyAddr, v.timeout)
	case proxy.SOCKS5:
		proxyURI = fmt.Sprintf("socks5://%s?timeout=%s", proxyAddr, v.timeout)
	default:
		return false
	}

	dialSocksProxy := socks.Dial(proxyURI)
	if dialSocksProxy == nil {
		return false
	}

	transport := &http.Transport{
		Dial:                dialSocksProxy,
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

	for k, v := range v.testHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
