package main

import (
	"log"

	"github.com/aredoff/proxygun/internal/parser"
	"github.com/aredoff/proxygun/internal/parser/providers"
	"github.com/aredoff/proxygun/internal/proxy"
	"github.com/aredoff/proxygun/internal/validator"
)

func main() {
	providers := []parser.Parser{
		providers.NewSSLProxiesProvider(),
		providers.NewUSProxyProvider(),
		providers.NewFreeProxyListProvider(),
		providers.NewCheckerProxyNetProvider(),
		providers.NewGithubTheSpeedXProvider(),
		providers.NewHideMyNameProvider(),
		providers.NewGithubMmpx12Provider(),
		providers.NewKuaidailiProvider(),
	}

	for _, provider := range providers {
		proxies, err := provider.Parse()
		if err != nil {
			log.Fatalf("Failed to parse provider %s: %v", provider.Name(), err)
		}

		val := validator.NewValidator()
		validProxies := ValidateProxiesConcurrent(val, proxies, 100)
		httpCount := 0
		socks4Count := 0
		socks5Count := 0
		for _, p := range validProxies {
			if p.Type == proxy.HTTP {
				httpCount++
			}
			if p.Type == proxy.SOCKS4 {
				socks4Count++
			}
			if p.Type == proxy.SOCKS5 {
				socks5Count++
			}
		}
		log.Printf("Found %d proxies and %d valid proxies (http: %d, socks4: %d, socks5: %d) from %s", len(proxies), len(validProxies), httpCount, socks4Count, socks5Count, provider.Name())
	}
}

func ValidateProxiesConcurrent(val *validator.Validator, proxies []*proxy.Proxy, workers int) []*proxy.Proxy {
	if workers <= 0 {
		workers = 10
	}

	jobs := make(chan *proxy.Proxy, len(proxies))
	results := make(chan *proxy.Proxy, len(proxies))

	for w := 0; w < workers; w++ {
		go func() {
			for p := range jobs {
				p, ok := val.ValidateAndDetectType(p)
				if ok {
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
