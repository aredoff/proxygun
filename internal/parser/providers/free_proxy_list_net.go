package providers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aredoff/proxygun/internal/proxy"
)

// https://free-proxy-list.net/ru/ssl-proxy.html
type SSLProxiesProvider struct {
	client *http.Client
}

func NewSSLProxiesProvider() *SSLProxiesProvider {
	return &SSLProxiesProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

func (p *SSLProxiesProvider) Name() string {
	return "SSLProxies"
}

func (p *SSLProxiesProvider) Parse() ([]*proxy.Proxy, error) {
	req, err := http.NewRequest("GET", "https://free-proxy-list.net/ru/ssl-proxy.html", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

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

	// Look for proxy table - may have different selectors
	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		tds := s.Find("td")
		if tds.Length() >= 7 { // SSL Proxies usually has 8 columns
			host := strings.TrimSpace(tds.Eq(0).Text())
			portStr := strings.TrimSpace(tds.Eq(1).Text())
			httpsSupport := strings.TrimSpace(tds.Eq(6).Text()) // HTTPS column

			if host == "" || portStr == "" {
				return
			}

			port, err := strconv.Atoi(portStr)
			if err != nil || port <= 0 {
				return
			}

			proxyType := proxy.HTTP
			if strings.ToLower(httpsSupport) == "yes" {
				proxyType = proxy.HTTP // HTTPS is also HTTP proxy
			}

			proxies = append(proxies, &proxy.Proxy{
				Host: host,
				Port: port,
				Type: proxyType,
			})
		}
	})

	return proxies, nil
}
