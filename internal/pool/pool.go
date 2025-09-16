package pool

import (
	"sync"

	"github.com/aredoff/proxygun/internal/proxy"
)

func NewProxyWithStats(p *proxy.Proxy) *proxy.ProxyWithStats {
	return proxy.NewProxyWithStats(p)
}

type Pool struct {
	proxies     []*proxy.ProxyWithStats          //Main pool of proxies
	badProxies  map[string]*proxy.ProxyWithStats //Pool of bad proxies
	freePool    []*proxy.ProxyWithStats          //Pool of free proxies
	current     int
	maxSize     int
	minRequests int
	mu          sync.RWMutex
}

func NewPool(maxSize int) *Pool {
	return &Pool{
		proxies:     make([]*proxy.ProxyWithStats, 0, maxSize),
		badProxies:  make(map[string]*proxy.ProxyWithStats),
		freePool:    make([]*proxy.ProxyWithStats, 0),
		maxSize:     maxSize,
		minRequests: 10,
	}
}

func (p *Pool) Add(proxy *proxy.Proxy) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	proxyKey := proxy.String()
	if _, exists := p.badProxies[proxyKey]; exists {
		return false
	}

	for _, p := range p.proxies {
		if p.Proxy.String() == proxyKey {
			return false
		}
	}

	proxyWithStats := NewProxyWithStats(proxy)

	if len(p.proxies) < p.maxSize {
		p.proxies = append(p.proxies, proxyWithStats)
		return true
	}

	p.freePool = append(p.freePool, proxyWithStats)
	return true
}

func (p *Pool) fillProxiesFromFree() {
	p.mu.Lock()
	defer p.mu.Unlock()

	moved := 0
	if len(p.proxies) < p.maxSize {
		for len(p.freePool) > 0 && len(p.proxies) < p.maxSize {
			p.proxies = append(p.proxies, p.freePool[0])
			p.freePool = p.freePool[1:]
			moved++
		}
		if moved > 0 && len(p.proxies) > 0 {
			if p.current >= len(p.proxies) {
				p.current = 0
			}
		}
	}
}

// FillFromFree moves proxies from free pool to main pool if needed
func (p *Pool) FillFromFree() {
	p.fillProxiesFromFree()
}

func (p *Pool) Next() *proxy.ProxyWithStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.proxies) == 0 {
		return nil
	}

	if len(p.proxies) < p.maxSize {
		p.mu.RUnlock()
		p.fillProxiesFromFree()
		p.mu.RLock()

		if len(p.proxies) == 0 {
			return nil
		}
	}

	proxy := p.proxies[p.current]
	p.current = (p.current + 1) % len(p.proxies)
	return proxy
}

func (p *Pool) Remove(proxy *proxy.Proxy) {
	p.mu.Lock()
	defer p.mu.Unlock()

	proxyKey := proxy.String()

	for i, px := range p.proxies {
		if px.Proxy.String() == proxyKey {
			p.proxies = append(p.proxies[:i], p.proxies[i+1:]...)
			if p.current >= len(p.proxies) && len(p.proxies) > 0 {
				p.current = 0
			}
			break
		}
	}

	if len(p.freePool) > 0 && len(p.proxies) < p.maxSize {
		p.proxies = append(p.proxies, p.freePool[0])
		p.freePool = p.freePool[1:]
	}
}

func (p *Pool) MoveToBad(proxy *proxy.Proxy) {
	p.mu.Lock()
	defer p.mu.Unlock()

	proxyKey := proxy.String()

	for i, px := range p.proxies {
		if px.Proxy.String() == proxyKey {
			p.badProxies[proxyKey] = px
			p.proxies = append(p.proxies[:i], p.proxies[i+1:]...)
			if p.current >= len(p.proxies) && len(p.proxies) > 0 {
				p.current = 0
			}
			break
		}
	}

	if len(p.freePool) > 0 && len(p.proxies) < p.maxSize {
		p.proxies = append(p.proxies, p.freePool[0])
		p.freePool = p.freePool[1:]
	}
}

func (p *Pool) CheckBadProxies() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := len(p.proxies) - 1; i >= 0; i-- {
		proxy := p.proxies[i]
		if proxy.Stats.IsBad(p.minRequests) {
			proxyKey := proxy.Proxy.String()
			p.badProxies[proxyKey] = proxy
			p.proxies = append(p.proxies[:i], p.proxies[i+1:]...)
		}
	}

	if p.current >= len(p.proxies) && len(p.proxies) > 0 {
		p.current = 0
	}

	for len(p.freePool) > 0 && len(p.proxies) < p.maxSize {
		p.proxies = append(p.proxies, p.freePool[0])
		p.freePool = p.freePool[1:]
	}
}

func (p *Pool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.proxies)
}

func (p *Pool) FreeSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.freePool)
}

func (p *Pool) NeedsProxies() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.maxSize - len(p.proxies)
}
