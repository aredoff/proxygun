package providers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aredoff/proxygun/internal/proxy"
)

type KuaidailiProvider struct {
	client *http.Client
}

func NewKuaidailiProvider() *KuaidailiProvider {
	return &KuaidailiProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

func (p *KuaidailiProvider) Name() string {
	return "Kuaidaili"
}

func (p *KuaidailiProvider) Parse() ([]*proxy.Proxy, error) {
	var allProxies []*proxy.Proxy

	for page := 1; page <= 3; page++ {
		proxies, err := p.parsePage(page)
		if err != nil {
			continue
		}
		allProxies = append(allProxies, proxies...)
	}

	return allProxies, nil
}

func (p *KuaidailiProvider) parsePage(page int) ([]*proxy.Proxy, error) {
	url := fmt.Sprintf("https://www.kuaidaili.com/free/inha/%d/", page)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var proxies []*proxy.Proxy

	doc.Find("#list table tbody tr").Each(func(i int, s *goquery.Selection) {
		tds := s.Find("td")
		if tds.Length() >= 4 {
			host := strings.TrimSpace(tds.Eq(0).Text())
			portStr := strings.TrimSpace(tds.Eq(1).Text())
			typeStr := strings.TrimSpace(tds.Eq(3).Text())

			if host == "" || portStr == "" {
				return
			}

			port, err := strconv.Atoi(portStr)
			if err != nil || port <= 0 {
				return
			}

			var proxyType proxy.Type
			switch strings.ToUpper(typeStr) {
			case "HTTP", "HTTPS":
				proxyType = proxy.HTTP
			case "SOCKS4":
				proxyType = proxy.SOCKS4
			case "SOCKS5":
				proxyType = proxy.SOCKS5
			default:
				proxyType = proxy.HTTP
			}

			if isValidIP(host) && isValidPort(portStr) {
				proxies = append(proxies, &proxy.Proxy{
					Host: host,
					Port: port,
					Type: proxyType,
				})
			}
		}
	})

	return proxies, nil
}
