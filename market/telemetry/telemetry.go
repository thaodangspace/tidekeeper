// Package telemetry defines infrastructure-neutral market calculation events.
package telemetry

import "time"

// Event is emitted at an authoritative calculation service boundary.
type Event struct {
	Operation     string
	DailyTideID   string
	Sector        string
	MappingID     string
	Status        string
	CoveredWeight int64
	Checksum      string
	Duration      time.Duration
	Err           error
}

// Recorder can bridge calculation events to structured logs and metrics.
type Recorder interface {
	RecordMarketCalculation(Event)
}

// NopRecorder is the default recorder when observability is not configured.
type NopRecorder struct{}

func (NopRecorder) RecordMarketCalculation(Event) {}
