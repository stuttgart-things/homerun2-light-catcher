package catcher

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	homerun "github.com/stuttgart-things/homerun-library/v4"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/dashboard"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/mock"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/models"
)

func TestStreamEntryTime(t *testing.T) {
	got, err := streamEntryTime("1789100802521-0")
	if err != nil {
		t.Fatalf("streamEntryTime: %v", err)
	}
	if want := time.UnixMilli(1789100802521); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, id := range []string{"", "abc-0", "-5-0", "not-an-id"} {
		if _, err := streamEntryTime(id); err == nil {
			t.Errorf("streamEntryTime(%q): expected error", id)
		}
	}
}

// lightHandlerFixture runs LightHandler against the embedded WLED mock with a
// profile that matches every info message.
func lightHandlerFixture(t *testing.T, maxAge time.Duration) (MessageHandler, *dashboard.EventTracker) {
	t.Helper()
	srv := httptest.NewServer(mock.NewServer("test", "abc1234", "2026-01-01").Handler())
	t.Cleanup(srv.Close)

	profileYAML := fmt.Sprintf(`effects:
  info:
    systems: ["*"]
    severity: [info]
    fx: Solid
    color: blue
    endpoint: %s
`, srv.URL)
	path := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.WriteFile(path, []byte(profileYAML), 0644); err != nil {
		t.Fatal(err)
	}

	tracker := dashboard.NewEventTracker()
	return LightHandler(path, maxAge, tracker), tracker
}

func caughtAt(pitched time.Time, eventTimestamp string) models.CaughtMessage {
	return models.CaughtMessage{
		Message:  homerun.Message{System: "github", Severity: "info", Timestamp: eventTimestamp},
		ObjectID: "msg",
		StreamID: fmt.Sprintf("%d-0", pitched.UnixMilli()),
	}
}

func TestLightHandler_MessageAge(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		maxAge  time.Duration
		msg     models.CaughtMessage
		trigger bool
	}{
		{"just pitched", time.Minute, caughtAt(now, now.Format(time.RFC3339)), true},
		{"pitched longer ago than max age", time.Minute, caughtAt(now.Add(-2*time.Minute), now.Format(time.RFC3339)), false},
		{"old RFC3339 backlog is skipped", time.Minute, caughtAt(now.Add(-7*time.Hour), now.Add(-7*time.Hour).Format(time.RFC3339)), false},
		{"old Unix-seconds backlog is skipped", time.Minute, caughtAt(now.Add(-7*time.Hour), strconv.FormatInt(now.Add(-7*time.Hour).Unix(), 10)), false},
		// git-pitcher sets the timestamp to the GitHub event's creation time,
		// minutes before it polls and pitches it.
		{"old event time but just pitched", time.Minute, caughtAt(now, now.Add(-25*time.Minute).Format(time.RFC3339)), true},
		{"empty timestamp field is irrelevant", time.Minute, caughtAt(now, ""), true},
		{"unparseable timestamp field is irrelevant", time.Minute, caughtAt(now, "yesterday"), true},
		{"Redis clock slightly ahead", time.Minute, caughtAt(now.Add(2*time.Second), ""), true},
		{"max age 0 disables the check", 0, caughtAt(now.Add(-7*time.Hour), ""), true},
		{"unparseable stream ID is allowed", time.Minute, models.CaughtMessage{Message: homerun.Message{System: "github", Severity: "info"}, StreamID: "bogus"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, tracker := lightHandlerFixture(t, tt.maxAge)
			handler(tt.msg)
			if got := tracker.Count() == 1; got != tt.trigger {
				t.Errorf("triggered = %v, want %v", got, tt.trigger)
			}
		})
	}
}

func TestSeverityToLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ERROR", "ERROR"},
		{"error", "ERROR"},
		{"WARNING", "WARN"},
		{"warning", "WARN"},
		{"INFO", "INFO"},
		{"SUCCESS", "INFO"},
		{"DEBUG", "DEBUG"},
	}

	for _, tt := range tests {
		level := severityToLevel(tt.input)
		if level.String() != tt.expected {
			t.Errorf("severityToLevel(%q) = %s, want %s", tt.input, level.String(), tt.expected)
		}
	}
}
