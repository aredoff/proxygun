package providers

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aredoff/proxygun/internal/proxy"
)

type GithubTheSpeedXProvider struct {
	client *http.Client
	urls   []string
}

func NewGithubTheSpeedXProvider() *GithubTheSpeedXProvider {
	return &GithubTheSpeedXProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
		urls: []string{
			"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt",
			"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/socks4.txt",
			"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/socks5.txt",
		},
	}
}

func (p *GithubTheSpeedXProvider) Name() string {
	return "GithubTheSpeedX"
}

func (p *GithubTheSpeedXProvider) Parse() ([]*proxy.Proxy, error) {
	var allProxies []*proxy.Proxy

	for _, url := range p.urls {
		proxies, err := p.parseURL(url)
		if err != nil {
			continue // Skip failed URLs
		}
		allProxies = append(allProxies, proxies...)
	}

	if len(allProxies) == 0 {
		return nil, fmt.Errorf("no proxies found from any source")
	}

	return allProxies, nil
}

func (p *GithubTheSpeedXProvider) parseURL(url string) ([]*proxy.Proxy, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var proxies []*proxy.Proxy
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
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

	return proxies, scanner.Err()
}
