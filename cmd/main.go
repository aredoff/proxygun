package main

import (
	"fmt"
	"log"
	"time"

	"github.com/aredoff/proxygun"
)

func main() {
	config := proxygun.DefaultConfig()
	config.PoolSize = 20
	config.MaxRetries = 5

	client := proxygun.NewClient(config)
	defer client.Close()

	time.Sleep(5 * time.Second)

	resp, err := client.Get("https://httpbin.org/ip")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Stats: %+v\n", client.Stats())
}
