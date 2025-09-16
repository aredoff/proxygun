package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aredoff/proxygun/internal/proxy"
)

type CheckerProxyNetProvider struct {
	client *http.Client
}

type checkerProxyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Date      string   `json:"date"`
		ProxyList []string `json:"proxyList"`
	} `json:"data"`
}

func NewCheckerProxyNetProvider() *CheckerProxyNetProvider {
	return &CheckerProxyNetProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

func (p *CheckerProxyNetProvider) Name() string {
	return "CheckerProxyNet"
}

func (p *CheckerProxyNetProvider) Parse() ([]*proxy.Proxy, error) {
	url := fmt.Sprintf("https://api.checkerproxy.net/v1/landing/archive/%s", time.Now().Format("2006-01-02"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response checkerProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if !response.Success {
		return nil, fmt.Errorf("API error: %s", response.Message)
	}

	var proxies []*proxy.Proxy
	for _, proxyStr := range response.Data.ProxyList {
		parts := strings.Split(proxyStr, ":")
		if len(parts) != 2 {
			continue
		}

		host := strings.TrimSpace(parts[0])
		portStr := strings.TrimSpace(parts[1])

		if !isValidIP(host) || !isValidPort(portStr) {
			continue
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		// Use HTTP as default type, will be detected during validation
		proxies = append(proxies, &proxy.Proxy{
			Host: host,
			Port: port,
			Type: proxy.HTTP,
		})
	}

	return proxies, nil
}
