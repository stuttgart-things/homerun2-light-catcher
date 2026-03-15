package wled

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/dashboard"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/profile"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// SendToWLED loads the profile, matches an effect, and sends it to the WLED device.
func SendToWLED(profilePath, severity, system string, tracker *dashboard.EventTracker) {
	config, err := profile.LoadConfiguration(profilePath)
	if err != nil {
		slog.Error("failed to load profile", "error", err)
		return
	}

	effect, found := profile.MatchEffect(config, system, severity)
	if !found {
		slog.Warn("no matching effect", "system", system, "severity", severity)
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

	meta := EffectMeta{Severity: severity, System: system, Effect: effect.Fx, Color: effect.Color}
	if err := SendEffect(effect.Endpoint, fx, colors, meta); err != nil {
		slog.Error("failed to send WLED effect", "endpoint", effect.Endpoint, "error", err)
		return
	}

	slog.Info("WLED effect triggered",
		"fx", effect.Fx,
		"color", effect.Color,
		"endpoint", effect.Endpoint,
		"duration", effect.Duration,
		"system", system,
		"severity", severity,
	)

	if tracker != nil {
		tracker.Record(severity, system, effect.Fx, effect.Color, effect.Endpoint)
	}

	if effect.Duration > 0 {
		go func() {
			time.Sleep(time.Duration(effect.Duration) * time.Second)
			if err := TurnOff(effect.Endpoint); err != nil {
				slog.Error("failed to turn off WLED", "endpoint", effect.Endpoint, "error", err)
			} else {
				slog.Info("WLED light turned off", "endpoint", effect.Endpoint)
				if tracker != nil {
					tracker.RecordOff(effect.Endpoint)
				}
			}
		}()
	}
}

// EffectMeta carries context about what triggered the WLED effect.
// Real WLED devices ignore unknown fields; the mock uses them for display.
type EffectMeta struct {
	Severity string
	System   string
	Effect   string
	Color    string
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
