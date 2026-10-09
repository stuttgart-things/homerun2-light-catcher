package wled

import (
	"testing"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/dashboard"
)

// Tests for #84: the timeline carries the triggering message, and a replay
// plays an effect again even in quiet hours.

func quietAllDay(endpoint string) string {
	return `quietHours:
  from: "00:00"
  to: "23:59"
  allow: []
effects:
  ci:
    systems: ["*"]
    severity: [warning]
    fx: Breathe
    duration: 0
    color: orange
    endpoint: ` + endpoint + "\n"
}

func TestTrigger_RecordsTheSourceMessage(t *testing.T) {
	srv := mockServer(t)
	path := writeProfile(t, `effects:
  ci:
    systems: ["*"]
    severity: [warning]
    fx: Breathe
    duration: 0
    color: orange
    endpoint: `+srv.URL+"\n")
	tracker := dashboard.NewEventTracker()

	src := Source{Severity: "warning", System: "git", Tags: "github,pull_request", Title: "PR #1", Message: "opened", Author: "dev", URL: "https://x/1"}
	if !Trigger(path, src, tracker, TriggerOptions{}) {
		t.Fatal("expected the effect to play")
	}

	ev := tracker.Events()[0]
	if ev.Title != "PR #1" || ev.Message != "opened" || ev.Author != "dev" || ev.URL != "https://x/1" || ev.MessageTags != "github,pull_request" {
		t.Errorf("source not recorded: %+v", ev)
	}
	if ev.ID != 1 || ev.ReplayOf != 0 {
		t.Errorf("unexpected id/replayOf: %+v", ev)
	}
}

func TestTrigger_ReplayIgnoresQuietHours(t *testing.T) {
	srv := mockServer(t)
	path := writeProfile(t, quietAllDay(srv.URL))
	tracker := dashboard.NewEventTracker()
	src := Source{Severity: "warning", System: "git"}

	if Trigger(path, src, tracker, TriggerOptions{}) {
		t.Fatal("a caught message must honor quiet hours")
	}
	if !Trigger(path, src, tracker, TriggerOptions{IgnoreQuietHours: true, ReplayOf: 7}) {
		t.Fatal("a replay must ignore quiet hours")
	}
	if ev := tracker.Events()[0]; ev.ReplayOf != 7 {
		t.Errorf("expected replayOf 7, got %d", ev.ReplayOf)
	}
}

func TestTrigger_NoMatchReturnsFalse(t *testing.T) {
	srv := mockServer(t)
	path := writeProfile(t, quietAllDay(srv.URL))

	if Trigger(path, Source{Severity: "info", System: "x"}, nil, TriggerOptions{IgnoreQuietHours: true}) {
		t.Error("no effect matches info")
	}
}
