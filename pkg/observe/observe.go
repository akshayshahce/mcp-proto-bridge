package observe

// Package observe defines optional observability hooks for bridge decoding.

import (
	"time"

	bridgeerrors "github.com/akshay/mcp-proto-bridge/pkg/errors"
)

// Tracer receives stage start/finish notifications.
// Returned callback is invoked exactly once when the stage ends.
type Tracer interface {
	StartStage(stage bridgeerrors.Stage) func(err error)
}

// Metrics receives per-stage latency and success measurements.
type Metrics interface {
	ObserveStage(stage bridgeerrors.Stage, duration time.Duration, success bool)
}

// EventLogger receives structured decode lifecycle events.
type EventLogger interface {
	LogEvent(Event)
}

// Event is emitted when a stage starts or finishes.
type Event struct {
	Kind       string
	Stage      bridgeerrors.Stage
	Profile    string
	Target     string
	Duration   time.Duration
	Err        error
	Provenance Provenance
	Drift      *DriftSignal
	Timestamp  time.Time
}

// Provenance describes key decode execution decisions for observability.
type Provenance struct {
	ResolvedVersion    string
	VersionRuleMatched bool
	AppliedProfile     string
	ExtractorMode      string
	AutoRepairPasses   int
	PolicyOnToolError  string
	PolicyOnNoPayload  string
	ValidationPolicy   string
}

// DriftSignal is emitted when runtime inputs diverge from expected rules.
type DriftSignal struct {
	Type     string
	Message  string
	Metadata map[string]string
}

// Hooks bundles optional observability integrations.
type Hooks struct {
	Tracer      Tracer
	Metrics     Metrics
	EventLogger EventLogger
}
