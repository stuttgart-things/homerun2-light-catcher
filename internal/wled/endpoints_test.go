package wled

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/dashboard"
)

// Tests for endpoints: one effect, several WLED devices.

var ciError = Source{Severity: "error", System: "ci"}

func endpointsProfile(duration string, restore bool, endpoints ...string) string {
	var sb strings.Builder
	sb.WriteString("effects:\n  alert:\n    systems: [\"*\"]\n    severity: [error]\n    fx: Fireworks\n    color: red\n")
	sb.WriteString("    duration: " + duration + "\n")
	if restore {
		sb.WriteString("    restore: true\n")
	}
	sb.WriteString("    endpoints:\n")
	for _, e := range endpoints {
		sb.WriteString("      - " + e + "\n")
	}
	return sb.String()
}

func TestTrigger_SendsToEveryEndpoint(t *testing.T) {
	a, b := mockServer(t), mockServer(t)
	path := writeProfile(t, endpointsProfile("0", false, a.URL, b.URL, a.URL))
	tracker := dashboard.NewEventTracker()

	if !Trigger(path, ciError, tracker, TriggerOptions{}) {
		t.Fatal("expected the effect to play")
	}
	for _, srv := range []string{a.URL, b.URL} {
		if st := mockState(t, srv); !st.On || st.Seg[0].Fx != 42 {
			t.Errorf("%s: want Fireworks on, got on=%v fx=%d", srv, st.On, st.Seg[0].Fx)
		}
	}
	// One event per device, a duplicate endpoint counted once.
	events := tracker.Events()
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	seen := map[string]bool{}
	for _, ev := range events {
		seen[ev.Endpoint] = true
	}
	if !seen[a.URL] || !seen[b.URL] {
		t.Errorf("want one event per endpoint, got %+v", seen)
	}
}

// A device that hangs must not hold back the effect on the others, and one
// that answers 500 must not stop them either.
func TestTrigger_SlowAndFailingEndpointsDoNotBlockOthers(t *testing.T) {
	good := mockServer(t)

	release := make(chan struct{})
	hanging := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(hanging.Close)

	var failed atomic.Int32
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failed.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failing.Close)

	path := writeProfile(t, endpointsProfile("0", false, hanging.URL, failing.URL, good.URL))

	done := make(chan bool, 1)
	go func() { done <- Trigger(path, ciError, nil, TriggerOptions{}) }()

	deadline := time.Now().Add(2 * time.Second)
	for !mockState(t, good.URL).On {
		if time.Now().After(deadline) {
			t.Fatal("the good device did not get the effect while another one hangs")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if failed.Load() == 0 {
		t.Error("the failing device was not tried")
	}

	close(release) // the hanging device answers 503 from now on
	select {
	case played := <-done:
		if !played {
			t.Error("Trigger must report true when one device got the effect")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("Trigger did not return")
	}
}

func TestTrigger_AllEndpointsFailReturnsFalse(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failing.Close)
	path := writeProfile(t, endpointsProfile("0", false, failing.URL))

	if Trigger(path, ciError, nil, TriggerOptions{}) {
		t.Error("no device got the effect")
	}
}

// Turn-off is per device: a newer effect on one device keeps that one on,
// the other goes off after the first effect's duration.
func TestTrigger_TurnOffPerEndpoint(t *testing.T) {
	shortDurations(t)
	a, b := mockServer(t), mockServer(t)
	path := writeProfile(t, `effects:
  alert:
    systems: ["*"]
    severity: [error]
    fx: Fireworks
    color: red
    duration: 2
    endpoint: `+a.URL+`
    endpoints: [`+b.URL+`]
  long:
    systems: ["*"]
    severity: [info]
    fx: Twinkle
    color: blue
    duration: 20
    endpoint: `+b.URL+"\n")

	Trigger(path, ciError, nil, TriggerOptions{})
	Trigger(path, Source{Severity: "info", System: "ci"}, nil, TriggerOptions{}) // newer effect on b only
	time.Sleep(250 * time.Millisecond)

	if mockState(t, a.URL).On {
		t.Error("a must be off after the alert's duration")
	}
	if st := mockState(t, b.URL); !st.On || st.Seg[0].Fx != 17 {
		t.Errorf("b must keep the newer effect (Twinkle 17), got on=%v fx=%d", st.On, st.Seg[0].Fx)
	}
}

// Restore is per device: each one gets its own scene back.
func TestTrigger_RestorePerEndpoint(t *testing.T) {
	shortDurations(t)
	a, b := mockServer(t), mockServer(t)
	postRaw(t, a.URL, `{"on":true,"seg":[{"fx":9,"pal":11}]}`)
	postRaw(t, b.URL, `{"on":true,"seg":[{"fx":2,"pal":5}]}`)
	path := writeProfile(t, endpointsProfile("2", true, a.URL, b.URL))

	Trigger(path, ciError, nil, TriggerOptions{})
	time.Sleep(250 * time.Millisecond)

	if st := mockState(t, a.URL); !st.On || st.Seg[0].Fx != 9 || st.Seg[0].Pal != 11 {
		t.Errorf("a: want fx 9 pal 11 back, got %+v", st.Seg[0])
	}
	if st := mockState(t, b.URL); !st.On || st.Seg[0].Fx != 2 || st.Seg[0].Pal != 5 {
		t.Errorf("b: want fx 2 pal 5 back, got %+v", st.Seg[0])
	}
}
