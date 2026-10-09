package dashboard

import (
	"sync"
	"time"
)

// LightEvent records a triggered WLED effect.
type LightEvent struct {
	// ID is a sequence the replay endpoint addresses the event by (#84).
	ID        int    `json:"id"`
	Timestamp string `json:"timestamp"`
	Severity  string `json:"severity"`
	System    string `json:"system"`
	Effect    string `json:"effect"`
	Color     string `json:"color"`
	Endpoint  string `json:"endpoint"`
	On        bool   `json:"on"`
	// Tags are the rule tags the message matched on.
	Tags []string `json:"tags,omitempty"`

	// The message that triggered the effect, so the timeline shows what made
	// the strip light up (#84).
	Title       string `json:"title,omitempty"`
	Message     string `json:"message,omitempty"`
	Author      string `json:"author,omitempty"`
	URL         string `json:"url,omitempty"`
	MessageTags string `json:"messageTags,omitempty"`

	// ReplayOf is the ID of the event this one plays again, 0 for a new one.
	ReplayOf int `json:"replayOf,omitempty"`
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

// Record adds a light event. tags are the rule tags the message matched on.
func (t *EventTracker) Record(severity, system, effect, color, endpoint string, tags []string) {
	t.RecordEvent(LightEvent{
		Severity: severity,
		System:   system,
		Effect:   effect,
		Color:    color,
		Endpoint: endpoint,
		On:       true,
		Tags:     tags,
	})
}

// RecordEvent adds a light event with all its fields and returns its ID.
// Timestamp and ID are set here.
func (t *EventTracker) RecordEvent(ev LightEvent) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.count++
	ev.ID = t.count
	ev.Timestamp = time.Now().Format("15:04:05")
	t.events[(t.count-1)%100] = ev
	return ev.ID
}

// RecordOff adds an off event.
func (t *EventTracker) RecordOff(endpoint string) {
	t.RecordEvent(LightEvent{Endpoint: endpoint, On: false})
}

// Get returns the event with this ID while it is still in the buffer.
func (t *EventTracker) Get(id int) (LightEvent, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if id <= 0 || id > t.count || t.count-id >= 100 {
		return LightEvent{}, false
	}
	return t.events[(id-1)%100], true
}

// Events returns recent events (oldest first).
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
