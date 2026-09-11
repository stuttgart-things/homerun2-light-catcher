package wled

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/dashboard"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/mock"
)

func mockState(t *testing.T, url string) mock.WLEDState {
	t.Helper()
	resp, err := http.Get(url + "/json/state")
	if err != nil {
		t.Fatalf("GET state: %v", err)
	}
	defer resp.Body.Close()
	var state mock.WLEDState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	return state
}

// TestSendToWLED_OlderTimerDoesNotTurnOffNewerEffect replays the scoreboard
// case from #50: a short point flash followed by a longer match celebration.
// The point's timer fires while the celebration is running and must not turn
// the light off.
func TestSendToWLED_OlderTimerDoesNotTurnOffNewerEffect(t *testing.T) {
	orig := durationUnit
	durationUnit = 50 * time.Millisecond
	t.Cleanup(func() { durationUnit = orig })

	srv := httptest.NewServer(mock.NewServer("test", "abc1234", "2026-01-01").Handler())
	defer srv.Close()

	profileYAML := fmt.Sprintf(`effects:
  point:
    systems: [tabletennis]
    severity: [info]
    fx: Solid
    duration: 3
    color: blue
    endpoint: %[1]s
  match-won:
    systems: [tabletennis]
    severity: [success]
    fx: Fireworks
    duration: 10
    color: forest
    endpoint: %[1]s
`, srv.URL)
	path := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.WriteFile(path, []byte(profileYAML), 0644); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	at := func(units int) { time.Sleep(time.Until(start.Add(time.Duration(units) * durationUnit))) }

	SendToWLED(path, "info", "tabletennis", "", nil) // t=0: point, off at t=3
	at(2)
	SendToWLED(path, "success", "tabletennis", "", nil) // t=2: match won, off at t=12

	at(6) // well past the point's timer
	if state := mockState(t, srv.URL); !state.On {
		t.Fatal("light turned off by the older effect's timer while the newer effect was running")
	} else if state.Seg[0].Fx != 6 {
		t.Fatalf("expected Fireworks (fx 6) still running, got fx %d", state.Seg[0].Fx)
	}

	deadline := start.Add(40 * durationUnit)
	for mockState(t, srv.URL).On {
		if time.Now().After(deadline) {
			t.Fatal("newer effect was never turned off by its own timer")
		}
		time.Sleep(durationUnit / 5)
	}
}

func TestSendToWLED_TimerTurnsOffWhenNoNewerEffect(t *testing.T) {
	orig := durationUnit
	durationUnit = 20 * time.Millisecond
	t.Cleanup(func() { durationUnit = orig })

	srv := httptest.NewServer(mock.NewServer("test", "abc1234", "2026-01-01").Handler())
	defer srv.Close()

	profileYAML := fmt.Sprintf(`effects:
  info:
    systems: ["*"]
    severity: [info]
    fx: Solid
    duration: 1
    color: blue
    endpoint: %s
`, srv.URL)
	path := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.WriteFile(path, []byte(profileYAML), 0644); err != nil {
		t.Fatal(err)
	}

	SendToWLED(path, "info", "any", "", nil)
	if !mockState(t, srv.URL).On {
		t.Fatal("expected light on after effect")
	}

	deadline := time.Now().Add(50 * durationUnit)
	for mockState(t, srv.URL).On {
		if time.Now().After(deadline) {
			t.Fatal("light was never turned off")
		}
		time.Sleep(durationUnit / 4)
	}
}

func TestSendToWLED_TagsReachMatchAndTracker(t *testing.T) {
	srv := httptest.NewServer(mock.NewServer("test", "abc1234", "2026-01-01").Handler())
	defer srv.Close()

	profileYAML := fmt.Sprintf(`effects:
  tabletennis-point-a:
    systems: [tabletennis]
    severity: [info]
    tags: [transition=point, side=a]
    fx: Solid
    color: blue
    endpoint: %[1]s
  tabletennis-point-b:
    systems: [tabletennis]
    severity: [info]
    tags: [transition=point, side=b]
    fx: Solid
    color: red
    endpoint: %[1]s
  info:
    systems: ["*"]
    severity: [info]
    fx: DJ Light
    color: ocean
    endpoint: %[1]s
`, srv.URL)
	path := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.WriteFile(path, []byte(profileYAML), 0644); err != nil {
		t.Fatal(err)
	}

	tracker := dashboard.NewEventTracker()
	SendToWLED(path, "info", "tabletennis", "match=36c17b30,set=2,transition=point,side=b", tracker)

	state := mockState(t, srv.URL)
	if !state.On || len(state.Seg) != 1 || state.Seg[0].Col[0] != [3]int{255, 0, 0} {
		t.Fatalf("expected side b rule (red), got %+v", state)
	}

	events := tracker.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 tracked event, got %d", len(events))
	}
	if want := []string{"transition=point", "side=b"}; !slices.Equal(events[0].Tags, want) {
		t.Errorf("tracked tags = %v, want %v", events[0].Tags, want)
	}

	resp, err := http.Get(srv.URL + "/json/state/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Events []mock.Event `json:"events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	last := body.Events[len(body.Events)-1]
	if want := []string{"transition=point", "side=b"}; !slices.Equal(last.Tags, want) {
		t.Errorf("mock event tags = %v, want %v", last.Tags, want)
	}
}
