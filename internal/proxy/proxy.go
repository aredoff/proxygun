package proxy

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

type Type int

const (
	HTTP Type = iota
	SOCKS4
	SOCKS5
)

type Proxy struct {
	Host string
	Port int
	Type Type
}

func (p *Proxy) String() string {
	return fmt.Sprintf("%s:%d", p.Host, p.Port)
}

func (p *Proxy) URL() *url.URL {
	var scheme string
	switch p.Type {
	case HTTP:
		scheme = "http"
	case SOCKS4:
		scheme = "socks4"
	case SOCKS5:
		scheme = "socks5"
	}

	return &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port)),
	}
}

type ProxyWithStats struct {
	Proxy *Proxy
	Stats *Stats
}

func NewProxyWithStats(proxy *Proxy) *ProxyWithStats {
	return &ProxyWithStats{
		Proxy: proxy,
		Stats: &Stats{
			FirstUsed: time.Now(),
		},
	}
}

func (p *ProxyWithStats) RecordSuccess() {
	p.Stats.TotalRequests++
	p.Stats.SuccessRequests++
	p.Stats.LastUsed = time.Now()
}

func (p *ProxyWithStats) RecordFailure() {
	p.Stats.TotalRequests++
	p.Stats.FailedRequests++
	p.Stats.LastUsed = time.Now()
}
