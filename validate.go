package proxygun

import (
	"sync"

	"github.com/aredoff/proxygun/internal/proxy"
	"github.com/aredoff/proxygun/internal/validator"
)

// func ValidateProxiesConcurrent(proxies []*proxy.Proxy, workers int) []*proxy.Proxy {
// 	if workers <= 0 {
// 		workers = 10
// 	}

// 	jobs := make(chan *proxy.Proxy, len(proxies))
// 	results := make(chan *proxy.Proxy, len(proxies))

// 	for w := 0; w < workers; w++ {
// 		go func() {
// 			for p := range jobs {
// 				if v.ValidateProxy(p) {
// 					results <- p
// 				} else {
// 					results <- nil
// 				}
// 			}
// 		}()
// 	}

// 	for _, p := range proxies {
// 		jobs <- p
// 	}
// 	close(jobs)

// 	validProxies := make([]*proxy.Proxy, 0)
// 	processed := 0
// 	for i := 0; i < len(proxies); i++ {
// 		if result := <-results; result != nil {
// 			validProxies = append(validProxies, result)
// 		}
// 		processed++
// 	}

// 	return validProxies
// }

func ValidateProxiesConcurrentStream(v *validator.Validator, proxies []*proxy.Proxy, workers int, validChan chan<- *proxy.Proxy) {
	if workers <= 0 {
		workers = 10
	}
	if workers > 50 {
		workers = 50
	}

	jobs := make(chan *proxy.Proxy, len(proxies))
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				if v.ValidateProxy(p) {
					validChan <- p
				}
			}
		}()
	}

	for _, p := range proxies {
		jobs <- p
	}
	close(jobs)

	wg.Wait()
}
