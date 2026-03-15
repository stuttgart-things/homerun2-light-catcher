package dashboard

import (
	"sync"
	"time"
)

// LightEvent records a triggered WLED effect.
type LightEvent struct {
	Timestamp string `json:"timestamp"`
	Severity  string `json:"severity"`
	System    string `json:"system"`
	Effect    string `json:"effect"`
	Color     string `json:"color"`
	Endpoint  string `json:"endpoint"`
	On        bool   `json:"on"`
}

// EventTracker records recent light events in a ring buffer.
type EventTracker struct {
	mu     sync.RWMutex
	events [100]LightEvent
	count  int
}

// NewEventTracker creates a new event tracker.
func NewEventTracker() *EventTracker {
	return &EventTracker{}
}

// Record adds a light event.
func (t *EventTracker) Record(severity, system, effect, color, endpoint string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events[t.count%100] = LightEvent{
		Timestamp: time.Now().Format("15:04:05"),
		Severity:  severity,
		System:    system,
		Effect:    effect,
		Color:     color,
		Endpoint:  endpoint,
		On:        true,
	}
	t.count++
}

// RecordOff adds an off event.
func (t *EventTracker) RecordOff(endpoint string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events[t.count%100] = LightEvent{
		Timestamp: time.Now().Format("15:04:05"),
		Endpoint:  endpoint,
		On:        false,
	}
	t.count++
}

// Events returns recent events (newest first).
func (t *EventTracker) Events() []LightEvent {
	t.mu.RLock()
	defer t.mu.RUnlock()

	total := min(t.count, 100)
	result := make([]LightEvent, total)
	start := 0
	if t.count > 100 {
		start = t.count % 100
	}
	for i := 0; i < total; i++ {
		result[i] = t.events[(start+i)%100]
	}
	return result
}

// Count returns total events recorded.
func (t *EventTracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.count
}
