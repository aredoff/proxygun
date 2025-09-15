package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/aredoff/proxygun"
)

func main() {
	log.Println("Testing custom fallback transport...")

	// Test 1: With default fallback transport (http.DefaultTransport)
	log.Println("\n=== Test 1: With default fallback transport ===")
	config1 := proxygun.DefaultConfig()
	config1.PoolSize = 3
	config1.MaxRetries = 2
	// FallbackTransport уже установлен в http.DefaultTransport

	client1 := proxygun.NewProxyClient(config1)
	defer client1.Close()

	// Сразу делаем запрос, должен сработать fallback
	resp1, err1 := client1.Get("https://httpbin.org/ip")
	if err1 != nil {
		log.Printf("Test 1 failed: %v", err1)
	} else {
		defer resp1.Body.Close()
		body1, _ := io.ReadAll(resp1.Body)
		log.Printf("Test 1 success (via fallback): %s", body1)
	}

	// Test 2: With custom fallback transport
	log.Println("\n=== Test 2: With custom fallback transport ===")
	customTransport := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}

	config2 := proxygun.DefaultConfig()
	config2.PoolSize = 3
	config2.MaxRetries = 2
	config2.FallbackTransport = customTransport

	client2 := proxygun.NewProxyClient(config2)
	defer client2.Close()

	resp2, err2 := client2.Get("https://httpbin.org/ip")
	if err2 != nil {
		log.Printf("Test 2 failed: %v", err2)
	} else {
		defer resp2.Body.Close()
		body2, _ := io.ReadAll(resp2.Body)
		log.Printf("Test 2 success (via custom fallback): %s", body2)
	}

	// Test 3: Without fallback transport
	log.Println("\n=== Test 3: Without fallback transport ===")
	config3 := proxygun.DefaultConfig()
	config3.PoolSize = 3
	config3.MaxRetries = 2
	config3.FallbackTransport = nil // Отключаем fallback

	client3 := proxygun.NewProxyClient(config3)
	defer client3.Close()

	resp3, err3 := client3.Get("https://httpbin.org/ip")
	if err3 != nil {
		log.Printf("Test 3 expected failure (no fallback): %v", err3)
	} else {
		defer resp3.Body.Close()
		body3, _ := io.ReadAll(resp3.Body)
		log.Printf("Test 3 unexpected success: %s", body3)
	}

	// Test 4: Wait for proxies and test normal flow
	log.Println("\n=== Test 4: With working proxies ===")
	config4 := proxygun.DefaultConfig()
	config4.PoolSize = 3
	config4.MaxRetries = 2

	client4 := proxygun.NewProxyClient(config4)
	defer client4.Close()

	log.Println("Waiting for proxies...")
	for i := 0; i < 15; i++ {
		stats := client4.Stats()
		if stats["pool_size"].(int) > 0 {
			log.Printf("Proxies loaded! Stats: %+v", stats)
			break
		}
		time.Sleep(1 * time.Second)
	}

	resp4, err4 := client4.Get("https://httpbin.org/ip")
	if err4 != nil {
		log.Printf("Test 4 error: %v", err4)
	} else {
		defer resp4.Body.Close()
		body4, _ := io.ReadAll(resp4.Body)
		log.Printf("Test 4 success (via proxy): %s", body4)
		log.Printf("Final stats: %+v", client4.Stats())
	}

	fmt.Println("\nAll tests completed!")
}
