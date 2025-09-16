package proxy

import "time"

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
