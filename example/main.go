package main

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aredoff/proxygun"
)

func main() {
	config := proxygun.DefaultConfig()
	config.PoolSize = 10
	config.MaxRetries = 3
	config.ValidationWorkers = 60

	client := proxygun.NewClient(config)
	defer client.Close()

	log.Println("Waiting for proxies to be loaded...")

	// Wait until at least one proxy is loaded
	for i := 0; i < 60; i++ {
		stats := client.Stats()
		if stats["pool_size"].(int) > 0 {
			log.Printf("Proxies loaded! Stats: %+v", stats)
			break
		}
		time.Sleep(1 * time.Second)
		if i%5 == 0 {
			log.Printf("Still waiting... Stats: %+v", stats)
		}
	}

	log.Printf("Stats: %+v", client.Stats())

	resp, err := client.Get("https://api.ipify.org?format=json")
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
	fmt.Printf("Final Stats: %+v\n", client.Stats())
}
