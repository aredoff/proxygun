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
	Host     string
	Port     int
	Type     Type
	Username string
	Password string
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

	u := &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port)),
	}

	if p.Username != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}

	return u
}

type Stats struct {
	TotalRequests   int
	SuccessRequests int
	FailedRequests  int
	LastUsed        time.Time
	FirstUsed       time.Time
}

func (s *Stats) SuccessRate() float64 {
	if s.TotalRequests == 0 {
		return 0
	}
	return float64(s.SuccessRequests) / float64(s.TotalRequests)
}

func (s *Stats) FailureRate() float64 {
	return 1.0 - s.SuccessRate()
}

func (s *Stats) IsBad(minRequests int) bool {
	if s.TotalRequests < minRequests {
		return false
	}
	return s.FailureRate() > 0.7
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
