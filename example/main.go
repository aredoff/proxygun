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
	log.Println("ProxyRoundTripper example")

	config := proxygun.DefaultConfig()
	config.PoolSize = 5
	config.MaxRetries = 2
	config.ValidationWorkers = 20

	// Create ProxyRoundTripper
	rt := proxygun.NewProxyRoundTripper(config)
	defer rt.Close()

	// Create custom http.Client with ProxyRoundTripper
	client := &http.Client{
		Transport: rt,
		Timeout:   45 * time.Second,
	}

	log.Println("Waiting for proxies to be loaded...")

	// Wait for proxies
	for i := 0; i < 30; i++ {
		stats := rt.Stats()
		if stats["pool_size"].(int) > 0 {
			log.Printf("Proxies loaded! Stats: %+v", stats)
			break
		}
		time.Sleep(1 * time.Second)
	}

	// Make request
	resp, err := client.Get("https://httpbin.org/ip")
	if err != nil {
		log.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	fmt.Printf("Response: %s\n", body)
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Final Stats: %+v\n", rt.Stats())

	// Demonstrate that you can use the same RoundTripper with multiple clients
	client2 := &http.Client{
		Transport: rt,
		Timeout:   60 * time.Second,
	}

	log.Println("\nUsing the same RoundTripper with another client...")
	resp2, err := client2.Get("https://api.ipify.org?format=json")
	if err != nil {
		log.Printf("Second request error: %v", err)
	} else {
		defer resp2.Body.Close()
		body2, _ := io.ReadAll(resp2.Body)
		fmt.Printf("Second response: %s\n", body2)
	}
}
