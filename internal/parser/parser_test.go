package parser

import (
	"testing"
	"time"
)

// TestRotatingParser тестирует ротацию провайдеров
func TestRotatingParser(t *testing.T) {
	parser := NewRotatingParser()

	// Проверяем, что у нас есть провайдеры
	if len(parser.parsers) == 0 {
		t.Fatal("No parsers configured")
	}

	expectedProviders := []string{"SSLProxies", "USProxy", "FreeProxyList", "UKProxy"}
	if len(parser.parsers) != len(expectedProviders) {
		t.Fatalf("Expected %d providers, got %d", len(expectedProviders), len(parser.parsers))
	}

	// Тестируем ротацию
	usedProviders := make([]string, 0)
	for i := 0; i < len(expectedProviders)*2; i++ {
		// Парсим следующий провайдер
		_, err := parser.ParseNext()
		if err != nil {
			t.Logf("Provider failed (this may be expected): %v", err)
		}

		// Получаем имя использованного провайдера
		actualProviderName := parser.GetCurrentProviderName()
		usedProviders = append(usedProviders, actualProviderName)

		t.Logf("Iteration %d: used provider %s", i+1, actualProviderName)
	}

	// Проверяем, что мы прошли по всем провайдерам
	providerCount := make(map[string]int)
	for _, provider := range usedProviders {
		providerCount[provider]++
	}

	t.Logf("Provider usage counts: %v", providerCount)

	for _, expectedProvider := range expectedProviders {
		if count, exists := providerCount[expectedProvider]; !exists || count == 0 {
			t.Errorf("Provider %s was never used", expectedProvider)
		}
	}
}

// TestMultiParser тестирует MultiParser с ротацией
func TestMultiParser(t *testing.T) {
	parser := NewMultiParser()

	// Тестируем несколько вызовов ParseAll
	for i := 0; i < 8; i++ {
		proxies, errs := parser.ParseAll()

		actualProviderName := parser.GetCurrentProviderName()
		t.Logf("Call %d: used provider %s", i+1, actualProviderName)

		if len(errs) > 0 {
			t.Logf("Errors from %s: %v", actualProviderName, errs)
		} else {
			t.Logf("Success from %s: found %d proxies", actualProviderName, len(proxies))
		}
	}
}

// TestRotatingParser_LiveData тестирует ротацию с реальными данными
func TestRotatingParser_LiveData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live test in short mode")
	}

	parser := NewRotatingParser()
	totalProxies := 0
	successfulProviders := 0

	// Тестируем каждый провайдер по одному разу
	for i := 0; i < len(parser.parsers); i++ {
		start := time.Now()
		proxies, err := parser.ParseNext()
		duration := time.Since(start)

		actualProviderName := parser.GetCurrentProviderName()

		if err != nil {
			t.Logf("Provider %s failed in %v: %v", actualProviderName, duration, err)
		} else {
			t.Logf("Provider %s succeeded in %v: found %d proxies", actualProviderName, duration, len(proxies))
			totalProxies += len(proxies)
			successfulProviders++
		}
	}

	t.Logf("Summary: %d providers succeeded, %d total proxies found", successfulProviders, totalProxies)

	if successfulProviders == 0 {
		t.Error("No providers succeeded")
	}
}

// TestMultiParser_LiveData тестирует MultiParser с реальными данными
func TestMultiParser_LiveData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live test in short mode")
	}

	parser := NewMultiParser()

	// Делаем несколько запросов для проверки ротации
	results := make(map[string]int) // провайдер -> количество прокси

	for i := 0; i < 4; i++ {
		start := time.Now()
		proxies, errs := parser.ParseAll()
		duration := time.Since(start)

		providerName := parser.GetCurrentProviderName()

		if len(errs) > 0 {
			t.Logf("Call %d (%s) failed in %v: %v", i+1, providerName, duration, errs)
			results[providerName] = -1 // отмечаем как failed
		} else {
			t.Logf("Call %d (%s) succeeded in %v: found %d proxies", i+1, providerName, duration, len(proxies))
			results[providerName] = len(proxies)
		}

		// Небольшая пауза между запросами
		time.Sleep(1 * time.Second)
	}

	t.Logf("Final results: %v", results)

	successCount := 0
	for provider, count := range results {
		if count >= 0 {
			successCount++
			t.Logf("Provider %s: %d proxies", provider, count)
		}
	}

	if successCount == 0 {
		t.Error("No providers succeeded")
	}
}

// BenchmarkRotatingParser бенчмарк для ротационного парсера
func BenchmarkRotatingParser(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	parser := NewRotatingParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proxies, err := parser.ParseNext()
		if err != nil {
			b.Logf("Parse failed: %v", err)
		} else if len(proxies) == 0 {
			b.Logf("No proxies found")
		}
	}
}
