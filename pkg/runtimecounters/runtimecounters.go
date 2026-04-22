// Package runtimecounters provides a threadsafe in-memory runtime counter store.
package runtimecounters

import (
	"sync"
	"time"
)

// Collector stores named integer counters.
type Collector struct {
	mu       sync.RWMutex
	counters map[string]int64
}

// Snapshot is a point-in-time copy of all counters.
type Snapshot struct {
	Counters   map[string]int64
	RecordedAt time.Time
}

// New creates a new Collector.
func New() *Collector {
	return &Collector{counters: make(map[string]int64)}
}

// Inc increments a counter by 1.
func (c *Collector) Inc(name string) {
	c.Add(name, 1)
}

// Add increments a counter by delta.
func (c *Collector) Add(name string, delta int64) {
	if c == nil || name == "" || delta == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters[name] += delta
}

// Snapshot returns a copy of all counters.
func (c *Collector) Snapshot() Snapshot {
	if c == nil {
		return Snapshot{Counters: map[string]int64{}, RecordedAt: time.Now()}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]int64, len(c.counters))
	for key, value := range c.counters {
		out[key] = value
	}
	return Snapshot{Counters: out, RecordedAt: time.Now()}
}
