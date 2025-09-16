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
		Timeout:   15 * time.Second,
	}

	log.Info().Msg("Starting requests (will use fallback if no proxies available)...")

	for i := 0; i < 10000; i++ {
		time.Sleep(1 * time.Second)
		// Make request
		resp, err := client.Get("https://httpbin.org/ip")
		if err != nil {
			log.Error().Err(err).Msg("Error making request")
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error().Err(err).Msg("Error reading response")
			continue
		}

		fmt.Printf("Response: %s\n", body)
		fmt.Printf("Status: %s\n", resp.Status)
		fmt.Printf("Final Stats: %+v\n", rt.Stats())
	}
}
