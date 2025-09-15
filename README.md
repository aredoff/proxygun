# ProxyGun

Go library providing an HTTP RoundTripper with automatic proxy server pool management.

## Features

- Automatic proxy downloading from known sources
- Proxy validation through google.com requests
- Proxy rotation for each request
- Proxy statistics and automatic Bad Pool placement
- HTTP and SOCKS5 proxy support
- Automatic pool replenishment when needed

## Usage

### Option 1: Using ProxyClient (Recommended)

```go
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

    client := proxygun.NewProxyClient(config)
    defer client.Close()

    // Wait for proxy loading
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
```

### Option 2: Using ProxyRoundTripper with Custom Client

```go
package main

import (
    "net/http"
    "time"

    "github.com/aredoff/proxygun"
)

func main() {
    config := proxygun.DefaultConfig()
    
    // Create ProxyRoundTripper
    rt := proxygun.NewProxyRoundTripper(config)
    defer rt.Close()

    // Use with custom http.Client
    client := &http.Client{
        Transport: rt,
        Timeout:   45 * time.Second,
    }

    resp, err := client.Get("https://httpbin.org/ip")
    // Handle response...
}
```

## Configuration

```go
type Config struct {
    PoolSize          int                // Proxy pool size (default 50)
    MaxRetries        int                // Maximum retry attempts (default 3)
    RefreshInterval   time.Duration      // Proxy refresh interval (default 10 seconds)
    ValidationWorkers int                // Number of validation workers (default 30, max 50)
    BadProxyMaxAge    time.Duration      // Bad proxy retention time (default 24 hours)
    FallbackTransport http.RoundTripper  // Fallback transport when all proxies fail (default http.DefaultTransport)
    Logger            zerolog.Logger     // Logger for internal messages (default console logger)
}
```

### Fallback Transport

By default, if all proxies fail, the library will use `http.DefaultTransport` for direct connections. You can customize this behavior:

```go
config := proxygun.DefaultConfig()

// Use custom fallback transport
config.FallbackTransport = &http.Transport{
    MaxIdleConns:       10,
    IdleConnTimeout:    30 * time.Second,
    DisableCompression: true,
}

// Disable fallback (fail if no proxies work)
config.FallbackTransport = nil
```

### Logging

The library uses [zerolog](https://github.com/rs/zerolog) for structured logging. By default, it outputs to stderr with a console-friendly format:

```go
config := proxygun.DefaultConfig()

// Use custom logger
config.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

// Disable logging
config.Logger = zerolog.Nop()

// JSON logging
config.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
```

## Architecture

The library consists of the following components:

- `client.go` - main HTTP RoundTripper with proxy rotation
- `internal/proxy/` - structures for proxy representation and statistics
- `internal/pool/` - proxy pool management (main, free, bad)
- `internal/parser/` - parsers for proxy websites
- `internal/validator/` - proxy validator through test requests

## API

### ProxyRoundTripper (Core)
- `NewProxyRoundTripper(config *Config) *ProxyRoundTripper` - Creates a new RoundTripper
- `RoundTrip(req *http.Request) (*http.Response, error)` - Implements http.RoundTripper interface
- `Stats() map[string]interface{}` - Returns proxy pool statistics
- `Close() error` - Stops background workers

### ProxyClient (Convenience Wrapper)
- `NewProxyClient(config *Config) *ProxyClient` - Creates a wrapped http.Client
- All standard http.Client methods (Get, Post, Do, etc.)
- `Stats() map[string]interface{}` - Returns proxy pool statistics
- `Close() error` - Stops background workers

### Legacy Compatibility
- `NewClient(config *Config) *http.Client` - Returns standard http.Client with ProxyRoundTripper

## Proxy Sources

- https://www.sslproxies.org
- https://www.us-proxy.org
- https://free-proxy-list.net/uk-proxy.html
- https://free-proxy-list.net

## License

MIT
