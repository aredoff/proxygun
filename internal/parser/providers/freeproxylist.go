package providers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aredoff/proxygun/internal/proxy"
)

// FreeProxyListProvider - specialized provider for free-proxy-list.net
type FreeProxyListProvider struct {
	client *http.Client
}

func NewFreeProxyListProvider() *FreeProxyListProvider {
	return &FreeProxyListProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

func (p *FreeProxyListProvider) Name() string {
	return "FreeProxyList"
}

func (p *FreeProxyListProvider) Parse() ([]*proxy.Proxy, error) {
	req, err := http.NewRequest("GET", "https://free-proxy-list.net", nil)
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

	// Free Proxy List uses id="proxylisttable"
	doc.Find("#proxylisttable tbody tr").Each(func(i int, s *goquery.Selection) {
		tds := s.Find("td")
		if tds.Length() >= 7 {
			host := strings.TrimSpace(tds.Eq(0).Text())
			portStr := strings.TrimSpace(tds.Eq(1).Text())
			httpsSupport := strings.TrimSpace(tds.Eq(6).Text())

			if host == "" || portStr == "" {
				return
			}

			port, err := strconv.Atoi(portStr)
			if err != nil || port <= 0 {
				return
			}

			proxyType := proxy.HTTP
			if strings.ToLower(httpsSupport) == "yes" {
				proxyType = proxy.HTTP
			}

			proxies = append(proxies, &proxy.Proxy{
				Host: host,
				Port: port,
				Type: proxyType,
			})
		}
	})

	// Additionally search in other possible tables
	if len(proxies) == 0 {
		doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
			tds := s.Find("td")
			if tds.Length() >= 2 {
				host := strings.TrimSpace(tds.Eq(0).Text())
				portStr := strings.TrimSpace(tds.Eq(1).Text())

				if isValidIP(host) && isValidPort(portStr) {
					port, _ := strconv.Atoi(portStr)
					proxies = append(proxies, &proxy.Proxy{
						Host: host,
						Port: port,
						Type: proxy.HTTP,
					})
				}
			}
		})
	}

	return proxies, nil
}
