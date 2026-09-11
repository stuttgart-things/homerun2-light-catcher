package wled

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/dashboard"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/profile"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// durationUnit is the unit of an effect's duration; tests shorten it.
var durationUnit = time.Second

// device tracks the effects sent to one WLED endpoint.
type device struct {
	// mu serializes effect and turn-off requests to the endpoint, so a
	// turn-off cannot interleave with a newer effect being sent.
	mu sync.Mutex
	// generation is bumped on every effect sent successfully.
	generation uint64
}

var (
	devicesMu sync.Mutex
	devices   = map[string]*device{}
)

func deviceFor(endpoint string) *device {
	devicesMu.Lock()
	defer devicesMu.Unlock()
	d, ok := devices[endpoint]
	if !ok {
		d = &device{}
		devices[endpoint] = d
	}
	return d
}

// SendToWLED loads the profile, matches an effect, and sends it to the WLED device.
// tags is the message's comma-separated tags field.
func SendToWLED(profilePath, severity, system, tags string, tracker *dashboard.EventTracker) {
	config, err := profile.LoadConfiguration(profilePath)
	if err != nil {
		slog.Error("failed to load profile", "error", err)
		return
	}

	effect, found := profile.MatchEffect(config, system, severity, tags)
	if !found {
		slog.Warn("no matching effect", "system", system, "severity", severity, "tags", tags)
		return
	}

	colors, err := profile.GetColor(effect.Color)
	if err != nil {
		slog.Error("failed to resolve color", "color", effect.Color, "error", err)
		return
	}

	fx, ok := profile.FxMap[effect.Fx]
	if !ok {
		slog.Error("unknown effect", "fx", effect.Fx)
		return
	}

	meta := EffectMeta{Severity: severity, System: system, Effect: effect.Fx, Color: effect.Color, Tags: effect.Tags}
	dev := deviceFor(effect.Endpoint)
	dev.mu.Lock()
	if err := SendEffect(effect.Endpoint, fx, colors, meta); err != nil {
		dev.mu.Unlock()
		slog.Error("failed to send WLED effect", "endpoint", effect.Endpoint, "error", err)
		return
	}
	dev.generation++
	generation := dev.generation
	dev.mu.Unlock()

	slog.Info("WLED effect triggered",
		"fx", effect.Fx,
		"color", effect.Color,
		"endpoint", effect.Endpoint,
		"duration", effect.Duration,
		"system", system,
		"severity", severity,
		"matched_tags", effect.Tags,
	)

	if tracker != nil {
		tracker.Record(severity, system, effect.Fx, effect.Color, effect.Endpoint, effect.Tags)
	}

	if effect.Duration > 0 {
		time.AfterFunc(time.Duration(effect.Duration)*durationUnit, func() {
			turnOffIfCurrent(dev, effect.Endpoint, generation, tracker)
		})
	}
}

// turnOffIfCurrent turns the endpoint off unless a newer effect has been sent
// to it since the effect with the given generation.
func turnOffIfCurrent(dev *device, endpoint string, generation uint64, tracker *dashboard.EventTracker) {
	dev.mu.Lock()
	defer dev.mu.Unlock()

	if dev.generation != generation {
		slog.Debug("skipping WLED turn-off, a newer effect was sent", "endpoint", endpoint)
		return
	}

	if err := TurnOff(endpoint); err != nil {
		slog.Error("failed to turn off WLED", "endpoint", endpoint, "error", err)
		return
	}

	slog.Info("WLED light turned off", "endpoint", endpoint)
	if tracker != nil {
		tracker.RecordOff(endpoint)
	}
}

// EffectMeta carries context about what triggered the WLED effect.
// Real WLED devices ignore unknown fields; the mock uses them for display.
type EffectMeta struct {
	Severity string
	System   string
	Effect   string
	Color    string
	// Tags are the rule tags the message matched on.
	Tags []string
}

// SendEffect sends an effect payload to the WLED JSON API.
func SendEffect(endpoint string, fx int, colors [][3]int, meta EffectMeta) error {
	payload := map[string]any{
		"on": true,
		"seg": []map[string]any{
			{
				"fx":  fx,
				"sx":  128,
				"ix":  255,
				"col": colors,
			},
		},
		"_severity": meta.Severity,
		"_system":   meta.System,
		"_effect":   meta.Effect,
		"_color":    meta.Color,
	}
	if len(meta.Tags) > 0 {
		payload["_tags"] = meta.Tags
	}

	return postState(endpoint, payload)
}

// TurnOff sends an off command to the WLED device.
func TurnOff(endpoint string) error {
	return postState(endpoint, map[string]any{"on": false})
}

func postState(endpoint string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := httpClient.Post(endpoint+"/json/state", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("POST %s/json/state: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WLED returned %s", resp.Status)
	}

	return nil
}
