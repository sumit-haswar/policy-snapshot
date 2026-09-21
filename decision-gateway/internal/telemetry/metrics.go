package telemetry

import "time"

type Metrics interface {
	RequestCompleted(route, statusClass string, duration time.Duration)
}

type NoopMetrics struct{}

func (NoopMetrics) RequestCompleted(string, string, time.Duration) {}
