package proxygun

import (
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
)

const (
	MinimalRequestsToCheckBad = 10
)

type Config struct {
	PoolSize          int
	MaxRetries        int
	RefreshInterval   time.Duration
	ValidationWorkers int
	GoodCodes         []int
	ErrorsToDie       int
	FallbackTransport http.RoundTripper
	Logger            zerolog.Logger
}

func DefaultConfig() *Config {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	return &Config{
		PoolSize:          50,
		MaxRetries:        3,
		RefreshInterval:   10 * time.Second,
		ValidationWorkers: 30,
		GoodCodes:         []int{200, 201, 202, 203, 204, 205, 206, 300, 301, 302, 303, 304, 305, 306, 307, 308},
		ErrorsToDie:       4,
		FallbackTransport: http.DefaultTransport,
		Logger:            logger,
	}
}
