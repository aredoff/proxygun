package providers

import (
	"testing"
)

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		ip    string
		valid bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"203.0.113.1", true},
		{"255.255.255.255", true},
		{"0.0.0.0", true},
		{"256.1.1.1", false},       // > 255
		{"192.168.1", false},       // не хватает октета
		{"192.168.1.1.1", false},   // слишком много октетов
		{"", false},                // пустая строка
		{"abc.def.ghi.jkl", false}, // не числа
		{"192.168.-1.1", false},    // отрицательное число
		{"192.168.1.01", true},     // ведущий ноль (допустимо)
	}

	for _, test := range tests {
		t.Run(test.ip, func(t *testing.T) {
			result := isValidIP(test.ip)
			if result != test.valid {
				t.Errorf("isValidIP(%q) = %v, want %v", test.ip, result, test.valid)
			}
		})
	}
}

func TestIsValidPort(t *testing.T) {
	tests := []struct {
		port  string
		valid bool
	}{
		{"80", true},
		{"443", true},
		{"8080", true},
		{"3128", true},
		{"65535", true},  // максимальный порт
		{"1", true},      // минимальный порт
		{"0", false},     // неверный порт
		{"65536", false}, // слишком большой
		{"-1", false},    // отрицательный
		{"", false},      // пустая строка
		{"abc", false},   // не число
		{"80.5", false},  // дробное число
		{"080", true},    // с ведущим нулем (должно работать)
	}

	for _, test := range tests {
		t.Run(test.port, func(t *testing.T) {
			result := isValidPort(test.port)
			if result != test.valid {
				t.Errorf("isValidPort(%q) = %v, want %v", test.port, result, test.valid)
			}
		})
	}
}

// BenchmarkIsValidIP бенчмарк для функции валидации IP
func BenchmarkIsValidIP(b *testing.B) {
	testIPs := []string{
		"192.168.1.1",
		"10.0.0.1",
		"256.1.1.1",       // invalid
		"abc.def.ghi.jkl", // invalid
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ip := range testIPs {
			isValidIP(ip)
		}
	}
}

// BenchmarkIsValidPort бенчмарк для функции валидации порта
func BenchmarkIsValidPort(b *testing.B) {
	testPorts := []string{
		"80",
		"8080",
		"65536", // invalid
		"abc",   // invalid
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, port := range testPorts {
			isValidPort(port)
		}
	}
}
