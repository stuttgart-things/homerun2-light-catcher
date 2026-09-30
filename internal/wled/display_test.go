package wled

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/profile"
)

// fakePanel is a led-catcher whose GET /display reports showing as set.
func fakePanel(t *testing.T) (*httptest.Server, *atomic.Bool) {
	t.Helper()
	var showing atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/display" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `{"showing":%t,"held":false,"idle":{"mode":"off"}}`, showing.Load())
	}))
	t.Cleanup(srv.Close)
	return srv, &showing
}

func shortAuto(t *testing.T) {
	t.Helper()
	orig := []time.Duration{displayPollInterval, displayStartGrace, displayUnreachable, autoMaxDuration}
	displayPollInterval, displayStartGrace, displayUnreachable, autoMaxDuration =
		10*time.Millisecond, 150*time.Millisecond, 150*time.Millisecond, 2*time.Second
	t.Cleanup(func() {
		displayPollInterval, displayStartGrace, displayUnreachable, autoMaxDuration = orig[0], orig[1], orig[2], orig[3]
	})
}

func autoProfile(t *testing.T, endpoint, display string) string {
	return writeProfile(t, `effects:
  alert:
    systems: ["*"]
    severity: [error]
    fx: Fireworks
    duration: auto
    color: red
    display: `+display+`
    endpoint: `+endpoint+"\n")
}

func TestDuration_YAML(t *testing.T) {
	for in, want := range map[string]profile.Duration{
		"duration: 3":      {Seconds: 3},
		"duration: auto":   {Auto: true},
		"duration: AUTO":   {Auto: true},
		`duration: "auto"`: {Auto: true},
		"":                 {},
	} {
		var e profile.Effect
		if err := yaml.Unmarshal([]byte(in), &e); err != nil || e.Duration != want {
			t.Errorf("%q: got %+v, %v; want %+v", in, e.Duration, err, want)
		}
	}
	var e profile.Effect
	if err := yaml.Unmarshal([]byte("duration: soon"), &e); err == nil {
		t.Error("duration: soon must be refused")
	}
}

// The light stays on exactly as long as the panel shows the message.
func TestAuto_FollowsThePanel(t *testing.T) {
	shortAuto(t)
	srv := mockServer(t)
	panel, showing := fakePanel(t)

	SendToWLED(autoProfile(t, srv.URL, panel.URL), "error", "ci", "", nil)
	showing.Store(true)
	time.Sleep(400 * time.Millisecond) // well past the start grace
	if !mockState(t, srv.URL).On {
		t.Fatal("the light must stay on while the panel shows the message")
	}

	showing.Store(false)
	time.Sleep(100 * time.Millisecond)
	if mockState(t, srv.URL).On {
		t.Fatal("the light must go off once the panel is done")
	}
}

// A panel that never starts showing (the led-catcher matched nothing, or
// dropped the message) must not keep the light on forever.
func TestAuto_PanelNeverStarts(t *testing.T) {
	shortAuto(t)
	srv := mockServer(t)
	panel, _ := fakePanel(t)

	SendToWLED(autoProfile(t, srv.URL, panel.URL), "error", "ci", "", nil)
	time.Sleep(300 * time.Millisecond)
	if mockState(t, srv.URL).On {
		t.Fatal("the light must go off after the start grace")
	}
}

func TestAuto_PanelUnreachable(t *testing.T) {
	shortAuto(t)
	srv := mockServer(t)

	SendToWLED(autoProfile(t, srv.URL, "http://127.0.0.1:1"), "error", "ci", "", nil)
	if !mockState(t, srv.URL).On {
		t.Fatal("the effect must still be shown")
	}
	time.Sleep(400 * time.Millisecond)
	if mockState(t, srv.URL).On {
		t.Fatal("an unreachable panel must end the effect after the fallback")
	}
}

// A held display (or a stuck panel) is capped.
func TestAuto_MaxDuration(t *testing.T) {
	shortAuto(t)
	autoMaxDuration = 300 * time.Millisecond
	srv := mockServer(t)
	panel, showing := fakePanel(t)
	showing.Store(true)

	SendToWLED(autoProfile(t, srv.URL, panel.URL), "error", "ci", "", nil)
	time.Sleep(500 * time.Millisecond)
	if mockState(t, srv.URL).On {
		t.Fatal("the light must go off at the max duration")
	}
}

// Without display and LED_CATCHER_URL, "auto" has nothing to follow: nothing
// is sent rather than a light that never goes off.
func TestAuto_NeedsADisplay(t *testing.T) {
	t.Setenv(ledCatcherURLEnv, "")
	srv := mockServer(t)

	SendToWLED(autoProfile(t, srv.URL, `""`), "error", "ci", "", nil)
	if mockState(t, srv.URL).Seg[0].Fx == 42 {
		t.Fatal("an auto effect without a display must not be sent")
	}
}

func TestAuto_DisplayFromEnv(t *testing.T) {
	shortAuto(t)
	srv := mockServer(t)
	panel, showing := fakePanel(t)
	t.Setenv(ledCatcherURLEnv, panel.URL+"/")

	SendToWLED(autoProfile(t, srv.URL, `""`), "error", "ci", "", nil)
	showing.Store(true)
	time.Sleep(300 * time.Millisecond)
	if !mockState(t, srv.URL).On {
		t.Fatal("LED_CATCHER_URL must be followed like display")
	}
	showing.Store(false)
	time.Sleep(100 * time.Millisecond)
	if mockState(t, srv.URL).On {
		t.Fatal("the light must go off once the panel is done")
	}
}
