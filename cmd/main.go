package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aredoff/proxygun"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup pretty console logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	config := proxygun.DefaultConfig()
	// Logger already configured in DefaultConfig()
	config.PoolSize = 20
	config.MaxRetries = 5

	client := proxygun.NewProxyClient(config)
	defer client.Close()

	time.Sleep(5 * time.Second)

	resp, err := client.Get("https://httpbin.org/ip")
	if err != nil {
		log.Error().Err(err).Msg("Request failed")
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Stats: %+v\n", client.Stats())
}
