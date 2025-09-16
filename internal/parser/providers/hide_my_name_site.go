package providers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aredoff/proxygun/internal/proxy"
)

type HideMyNameProvider struct {
	client *http.Client
}

func NewHideMyNameProvider() *HideMyNameProvider {
	return &HideMyNameProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

func (p *HideMyNameProvider) Name() string {
	return "HideMyName"
}

func (p *HideMyNameProvider) Parse() ([]*proxy.Proxy, error) {
	req, err := http.NewRequest("GET", "https://hide-my-name.site/proxy-list/", nil)
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

	// Parse proxy table rows
	doc.Find("table tbody tr, .proxy-list tr, .table tbody tr").Each(func(i int, s *goquery.Selection) {
		tds := s.Find("td")
		if tds.Length() >= 2 {
			host := strings.TrimSpace(tds.Eq(0).Text())
			portStr := strings.TrimSpace(tds.Eq(1).Text())

			if host == "" || portStr == "" {
				return
			}

			if !isValidIP(host) || !isValidPort(portStr) {
				return
			}

			port, err := strconv.Atoi(portStr)
			if err != nil || port <= 0 {
				return
			}

			// Use HTTP as default type, will be detected during validation
			proxies = append(proxies, &proxy.Proxy{
				Host: host,
				Port: port,
				Type: proxy.HTTP,
			})
		}
	})

	return proxies, nil
}
