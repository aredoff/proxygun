package providers

import (
	"testing"
	"time"

	"github.com/aredoff/proxygun/internal/proxy"
)

// Parser interface for providers (duplicate for tests)
type Parser interface {
	Parse() ([]*proxy.Proxy, error)
	Name() string
}

// TestAllProviders_LiveData tests all providers with real data
func TestAllProviders_LiveData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live tests in short mode")
	}

	providers := []struct {
		name     string
		provider Parser
	}{
		{"SSLProxies", NewSSLProxiesProvider()},
		{"USProxy", NewUSProxyProvider()},
		{"FreeProxyList", NewFreeProxyListProvider()},
		{"UKProxy", NewUKProxyProvider()},
	}

	totalProxies := 0
	validProxies := 0

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			t.Logf("Testing provider: %s", p.name)

			start := time.Now()
			proxies, err := p.provider.Parse()
			duration := time.Since(start)

			if err != nil {
				t.Errorf("Provider %s failed: %v", p.name, err)
				return
			}

			t.Logf("Provider %s: found %d proxies in %v", p.name, len(proxies), duration)
			totalProxies += len(proxies)

			if len(proxies) == 0 {
				t.Logf("Warning: Provider %s returned no proxies", p.name)
				return
			}

			// Check data quality
			valid := 0
			for i, proxy := range proxies {
				if proxy.Host == "" {
					t.Errorf("Proxy %d has empty host", i)
					continue
				}
				if proxy.Port <= 0 || proxy.Port > 65535 {
					t.Errorf("Proxy %d has invalid port: %d", i, proxy.Port)
					continue
				}
				// Check proxy type (usually HTTP)
				t.Logf("Proxy %d type: %v", i, proxy.Type)
				valid++
			}

			validProxies += valid
			successRate := float64(valid) / float64(len(proxies)) * 100
			t.Logf("Provider %s: %d/%d valid proxies (%.1f%%)", p.name, valid, len(proxies), successRate)

			// Show first few proxies for verification
			showCount := 3
			if len(proxies) < showCount {
				showCount = len(proxies)
			}

			for i := 0; i < showCount; i++ {
				proxy := proxies[i]
				t.Logf("Sample proxy %d: %s:%d (type: %v)", i+1, proxy.Host, proxy.Port, proxy.Type)
			}
		})
	}

	t.Logf("Total results: %d proxies found, %d valid (%.1f%%)",
		totalProxies, validProxies, float64(validProxies)/float64(totalProxies)*100)

	if totalProxies == 0 {
		t.Error("No proxies found from any provider")
	}
}

// TestSSLProxiesProvider_Live тестирует конкретно SSLProxies
func TestSSLProxiesProvider_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live test in short mode")
	}

	provider := NewSSLProxiesProvider()

	t.Logf("Testing live SSLProxies provider...")
	start := time.Now()
	proxies, err := provider.Parse()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("SSLProxies provider failed: %v", err)
	}

	t.Logf("SSLProxies: found %d proxies in %v", len(proxies), duration)

	if len(proxies) == 0 {
		t.Fatal("No proxies found from SSLProxies")
	}

	// Детальная проверка первых прокси
	for i, proxy := range proxies {
		if i >= 5 { // Проверяем только первые 5
			break
		}

		if !isValidIP(proxy.Host) {
			t.Errorf("Proxy %d has invalid IP: %s", i, proxy.Host)
		}
		if !isValidPort(string(rune(proxy.Port))) {
			t.Errorf("Proxy %d has invalid port: %d", i, proxy.Port)
		}

		t.Logf("Proxy %d: %s:%d", i+1, proxy.Host, proxy.Port)
	}
}

// TestUSProxyProvider_Live тестирует конкретно USProxy
func TestUSProxyProvider_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live test in short mode")
	}

	provider := NewUSProxyProvider()

	t.Logf("Testing live USProxy provider...")
	start := time.Now()
	proxies, err := provider.Parse()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("USProxy provider failed: %v", err)
	}

	t.Logf("USProxy: found %d proxies in %v", len(proxies), duration)

	if len(proxies) == 0 {
		t.Fatal("No proxies found from USProxy")
	}

	// Проверяем уникальность прокси
	seen := make(map[string]bool)
	duplicates := 0

	for _, proxy := range proxies {
		key := proxy.String()
		if seen[key] {
			duplicates++
		} else {
			seen[key] = true
		}
	}

	if duplicates > 0 {
		t.Logf("Warning: found %d duplicate proxies", duplicates)
	}

	t.Logf("USProxy: %d unique proxies", len(seen))
}

// TestFreeProxyListProvider_Live тестирует конкретно FreeProxyList
func TestFreeProxyListProvider_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live test in short mode")
	}

	provider := NewFreeProxyListProvider()

	t.Logf("Testing live FreeProxyList provider...")
	start := time.Now()
	proxies, err := provider.Parse()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("FreeProxyList provider failed: %v", err)
	}

	t.Logf("FreeProxyList: found %d proxies in %v", len(proxies), duration)

	if len(proxies) == 0 {
		t.Log("No proxies found from FreeProxyList (this may be expected)")
		return
	}

	// Анализируем распределение портов
	portCounts := make(map[int]int)
	for _, proxy := range proxies {
		portCounts[proxy.Port]++
	}

	t.Logf("Port distribution:")
	for port, count := range portCounts {
		if count > 1 {
			t.Logf("  Port %d: %d proxies", port, count)
		}
	}
}

// TestUKProxyProvider_Live тестирует конкретно UKProxy
func TestUKProxyProvider_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live test in short mode")
	}

	provider := NewUKProxyProvider()

	t.Logf("Testing live UKProxy provider...")
	start := time.Now()
	proxies, err := provider.Parse()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("UKProxy provider failed: %v", err)
	}

	t.Logf("UKProxy: found %d proxies in %v", len(proxies), duration)

	if len(proxies) == 0 {
		t.Log("No proxies found from UKProxy (this may be expected)")
		return
	}

	// Проверяем, что все прокси имеют корректный формат
	for i, proxy := range proxies {
		if proxy.Host == "" {
			t.Errorf("Proxy %d has empty host", i)
		}
		if proxy.Port == 0 {
			t.Errorf("Proxy %d has zero port", i)
		}
		if i < 3 { // Показываем первые 3
			t.Logf("UK Proxy %d: %s:%d", i+1, proxy.Host, proxy.Port)
		}
	}
}

// BenchmarkProviders бенчмарк для всех провайдеров
func BenchmarkProviders(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	providers := map[string]func() Parser{
		"SSLProxies":    func() Parser { return NewSSLProxiesProvider() },
		"USProxy":       func() Parser { return NewUSProxyProvider() },
		"FreeProxyList": func() Parser { return NewFreeProxyListProvider() },
		"UKProxy":       func() Parser { return NewUKProxyProvider() },
	}

	for name, createProvider := range providers {
		b.Run(name, func(b *testing.B) {
			provider := createProvider()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				proxies, err := provider.Parse()
				if err != nil {
					b.Fatalf("Provider %s failed: %v", name, err)
				}
				if len(proxies) == 0 {
					b.Logf("Warning: Provider %s returned no proxies", name)
				}
			}
		})
	}
}
