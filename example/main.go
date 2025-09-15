package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aredoff/proxygun"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup pretty console logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().Msg("ProxyRoundTripper example")

	config := proxygun.DefaultConfig()
	// Logger already configured in DefaultConfig()
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

	log.Info().Msg("Waiting for proxies to be loaded...")

	// Wait for proxies
	for i := 0; i < 30; i++ {
		stats := rt.Stats()
		if stats["pool_size"].(int) > 0 {
			log.Info().Interface("stats", stats).Msg("Proxies loaded!")
			break
		}
		time.Sleep(1 * time.Second)
	}

	// Make request
	resp, err := client.Get("https://httpbin.org/ip")
	if err != nil {
		log.Fatal().Err(err).Msg("Error making request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading response")
	}

	fmt.Printf("Response: %s\n", body)
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Final Stats: %+v\n", rt.Stats())

	// Demonstrate that you can use the same RoundTripper with multiple clients
	client2 := &http.Client{
		Transport: rt,
		Timeout:   60 * time.Second,
	}

	log.Info().Msg("Using the same RoundTripper with another client...")
	resp2, err := client2.Get("https://api.ipify.org?format=json")
	if err != nil {
		log.Error().Err(err).Msg("Second request error")
	} else {
		defer resp2.Body.Close()
		body2, _ := io.ReadAll(resp2.Body)
		fmt.Printf("Second response: %s\n", body2)
	}
}
