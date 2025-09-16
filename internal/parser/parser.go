package parser

import (
	"fmt"
	"sync"

	"github.com/aredoff/proxygun/internal/parser/providers"
	"github.com/aredoff/proxygun/internal/proxy"
)

// Parser interface for all proxy providers
type Parser interface {
	Parse() ([]*proxy.Proxy, error)
	Name() string
}

// RotatingParser rotates providers to avoid blocking
type RotatingParser struct {
	parsers    []Parser
	currentIdx int
	mu         sync.Mutex
}

// NewRotatingParser creates a new rotating parser
func NewRotatingParser() *RotatingParser {
	return &RotatingParser{
		parsers: []Parser{
			providers.NewSSLProxiesProvider(),
			providers.NewUSProxyProvider(),
			providers.NewFreeProxyListProvider(),
			providers.NewCheckerProxyNetProvider(),
			providers.NewGithubTheSpeedXProvider(),
			providers.NewHideMyNameProvider(),
			providers.NewGithubMmpx12Provider(),
			providers.NewKuaidailiProvider(),
		},
		currentIdx: 0,
	}
}

// ParseNext parses the next provider in rotation
func (p *RotatingParser) Next() ([]*proxy.Proxy, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.parsers) == 0 {
		return nil, fmt.Errorf("no parsers available")
	}

	parser := p.parsers[p.currentIdx]
	p.currentIdx = (p.currentIdx + 1) % len(p.parsers)

	return parser.Parse()
}

// GetCurrentProviderName returns the name of current provider
func (p *RotatingParser) GetCurrentProviderName() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.parsers) == 0 {
		return "none"
	}

	// Get the name of previous provider (which was used)
	prevIdx := (p.currentIdx - 1 + len(p.parsers)) % len(p.parsers)
	return p.parsers[prevIdx].Name()
}

// MultiParser combines multiple specialized providers (for compatibility)
type MultiParser struct {
	rotatingParser *RotatingParser
}

// NewMultiParser creates a new multi-parser with provider rotation
func NewMultiParser() *MultiParser {
	return &MultiParser{
		rotatingParser: NewRotatingParser(),
	}
}

// ParseAll performs parsing of one provider (with rotation)
func (p *MultiParser) Parse() ([]*proxy.Proxy, []error) {
	proxies, err := p.rotatingParser.Next()
	if err != nil {
		return nil, []error{err}
	}
	return proxies, nil
}

// GetCurrentProviderName returns the name of current provider
func (p *MultiParser) GetCurrentProviderName() string {
	return p.rotatingParser.GetCurrentProviderName()
}
