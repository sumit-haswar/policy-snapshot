package telemetry

import (
	"sync"
	"time"
)

type Metrics interface {
	RequestCompleted(route, statusClass string, duration time.Duration)
	PolicyRefreshCompleted(outcome string, duration time.Duration)
}

type NoopMetrics struct{}

func (NoopMetrics) RequestCompleted(string, string, time.Duration) {}
func (NoopMetrics) PolicyRefreshCompleted(string, time.Duration)   {}

type RequestObservation struct {
	Route       string
	StatusClass string
}

type MemoryMetrics struct {
	mu       sync.Mutex
	requests []RequestObservation
	refresh  []string
}

func (m *MemoryMetrics) RequestCompleted(route, statusClass string, _ time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, RequestObservation{Route: route, StatusClass: statusClass})
}

func (m *MemoryMetrics) PolicyRefreshCompleted(outcome string, _ time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refresh = append(m.refresh, outcome)
}

func (m *MemoryMetrics) Requests() []RequestObservation {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]RequestObservation(nil), m.requests...)
}

func (m *MemoryMetrics) RefreshOutcomes() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.refresh...)
}
