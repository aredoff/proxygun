package providers

import (
	"bufio"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aredoff/proxygun/internal/proxy"
)

type GithubMmpx12Provider struct {
	client *http.Client
}

func NewGithubMmpx12Provider() *GithubMmpx12Provider {
	return &GithubMmpx12Provider{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

func (p *GithubMmpx12Provider) Name() string {
	return "GithubMmpx12"
}

func (p *GithubMmpx12Provider) Parse() ([]*proxy.Proxy, error) {
	var allProxies []*proxy.Proxy

	urls := []struct {
		url       string
		proxyType proxy.Type
	}{
		{"https://raw.githubusercontent.com/mmpx12/proxy-list/master/http.txt", proxy.HTTP},
		{"https://raw.githubusercontent.com/mmpx12/proxy-list/master/https.txt", proxy.HTTP},
		{"https://raw.githubusercontent.com/mmpx12/proxy-list/master/socks4.txt", proxy.SOCKS4},
		{"https://raw.githubusercontent.com/mmpx12/proxy-list/master/socks5.txt", proxy.SOCKS5},
	}

	for _, urlInfo := range urls {
		proxies, err := p.parseURL(urlInfo.url, urlInfo.proxyType)
		if err != nil {
			continue
		}
		allProxies = append(allProxies, proxies...)
	}

	return allProxies, nil
}

func (p *GithubMmpx12Provider) parseURL(url string, proxyType proxy.Type) ([]*proxy.Proxy, error) {
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

		proxies = append(proxies, &proxy.Proxy{
			Host: host,
			Port: port,
			Type: proxyType,
		})
	}

	return proxies, scanner.Err()
}
