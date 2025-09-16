package validator

import (
	"context"
	"net"
	"net/http"

	"github.com/aredoff/proxygun/internal/proxy"
)

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
