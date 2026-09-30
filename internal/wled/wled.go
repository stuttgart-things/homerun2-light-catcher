package wled

import (
	"bytes"
	"encoding/json"
	"errors"
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
	// active is true from a successful effect until it is turned off or
	// restored: while it is, the device shows OUR effect, not its own state.
	active bool
	// baseline is the state read before the first effect of an active run,
	// written back when the last one ends with restore set. Captured only
	// while !active -- otherwise a second effect in quick succession would
	// "restore" the first one.
	baseline map[string]any
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

	if err = effect.Validate(); err != nil {
		slog.Error("invalid effect in profile", "fx", effect.Fx, "error", err)
		return
	}
	display := displayURL(effect.Display)
	if effect.Duration.Auto && display == "" {
		slog.Error("duration auto needs the led-catcher: set display or "+ledCatcherURLEnv, "fx", effect.Fx)
		return
	}

	colors, palette, err := resolveLook(effect)
	if err != nil {
		slog.Error("failed to resolve color", "color", effect.Color, "palette", effect.Palette, "endpoint", effect.Endpoint, "error", err)
		return
	}

	fx, fxSource, err := ResolveEffect(effect.Endpoint, effect.Fx)
	if err != nil {
		slog.Error("unknown effect", "fx", effect.Fx, "endpoint", effect.Endpoint, "error", err)
		return
	}
	if fxSource == SourceFallback {
		slog.Warn("effect list of the device unavailable, using the fallback table",
			"fx", effect.Fx, "id", fx, "endpoint", effect.Endpoint)
	}

	meta := EffectMeta{Severity: severity, System: system, Effect: effect.Fx, Color: effect.Color, Tags: effect.Tags}
	dev := deviceFor(effect.Endpoint)
	dev.mu.Lock()
	if effect.Restore && !dev.active {
		baseline, err := fetchRestoreState(effect.Endpoint)
		if err != nil {
			slog.Warn("cannot read the device state to restore, it will be switched off instead",
				"endpoint", effect.Endpoint, "error", err)
		}
		dev.baseline = baseline
	}
	if err := postState(effect.Endpoint, payloadFor(effect, fx, colors, palette, meta).body()); err != nil {
		dev.mu.Unlock()
		slog.Error("failed to send WLED effect", "endpoint", effect.Endpoint, "error", err)
		return
	}
	dev.active = true
	dev.generation++
	generation := dev.generation
	dev.mu.Unlock()

	slog.Info("WLED effect triggered",
		"fx", effect.Fx,
		"fx_id", fx,
		"fx_source", fxSource,
		"color", effect.Color,
		"endpoint", effect.Endpoint,
		"duration", effect.Duration.String(),
		"system", system,
		"severity", severity,
		"matched_tags", effect.Tags,
	)

	if tracker != nil {
		tracker.Record(severity, system, effect.Fx, effect.Color, effect.Endpoint, effect.Tags)
	}

	scheduleEnd(effect, display, dev, generation, tracker)
}

// scheduleEnd arranges for the effect with the given generation to end: after
// its duration in seconds, or with duration auto once the led-catcher's panel
// at display is done. A duration of 0 leaves the effect on.
func scheduleEnd(effect profile.Effect, display string, dev *device, generation uint64, tracker *dashboard.EventTracker) {
	switch {
	case effect.Duration.Auto:
		go func() {
			reason := followDisplay(display, dev, generation)
			slog.Debug("auto duration over", "endpoint", effect.Endpoint, "display", display, "reason", reason)
			turnOffIfCurrent(dev, effect.Endpoint, generation, effect.Restore, tracker)
		}()
	case effect.Duration.Seconds > 0:
		time.AfterFunc(time.Duration(effect.Duration.Seconds)*durationUnit, func() {
			turnOffIfCurrent(dev, effect.Endpoint, generation, effect.Restore, tracker)
		})
	}
}

// turnOffIfCurrent ends the effect with the given generation unless a newer
// effect has been sent to the endpoint since: it writes the baseline back if
// the effect asked for restore and one was read, and switches the device off
// otherwise.
func turnOffIfCurrent(dev *device, endpoint string, generation uint64, restore bool, tracker *dashboard.EventTracker) {
	dev.mu.Lock()
	defer dev.mu.Unlock()

	if dev.generation != generation {
		slog.Debug("skipping WLED turn-off, a newer effect was sent", "endpoint", endpoint)
		return
	}

	if restore && dev.baseline != nil {
		if err := postState(endpoint, dev.baseline); err != nil {
			slog.Error("failed to restore WLED state", "endpoint", endpoint, "error", err)
			return
		}
		slog.Info("WLED state restored", "endpoint", endpoint)
	} else {
		if err := TurnOff(endpoint); err != nil {
			slog.Error("failed to turn off WLED", "endpoint", endpoint, "error", err)
			return
		}
		slog.Info("WLED light turned off", "endpoint", endpoint)
	}
	dev.active = false
	dev.baseline = nil
	if tracker != nil {
		tracker.RecordOff(endpoint)
	}
}

// resolveLook turns the effect's color and palette into what is sent: local
// colors (col) first, as before; a name that is not a local color is looked up
// as a device palette (pal); an explicit palette is always a device palette.
func resolveLook(effect profile.Effect) (colors [][3]int, palette *int, err error) {
	if effect.Color != "" {
		colors, err = profile.GetColor(effect.Color)
		if err != nil {
			if effect.Palette != "" {
				// Both set and the color is not local: that is a typo, not a
				// palette -- say so instead of guessing.
				return nil, nil, err
			}
			pal, perr := ResolvePalette(effect.Endpoint, effect.Color)
			if perr != nil {
				return nil, nil, fmt.Errorf("%w; and as a device palette: %w", err, perr)
			}
			return nil, &pal, nil
		}
	}
	if effect.Palette != "" {
		pal, perr := ResolvePalette(effect.Endpoint, effect.Palette)
		if perr != nil {
			return nil, nil, perr
		}
		palette = &pal
	}
	if colors == nil && palette == nil {
		return nil, nil, errors.New("effect sets neither color nor palette")
	}
	return colors, palette, nil
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
	p := effectPayload{fx: fx, colors: colors, speed: defaultSpeed, intensity: defaultIntensity, meta: meta}
	return postState(endpoint, p.body())
}

// TurnOff sends an off command to the WLED device.
func TurnOff(endpoint string) error {
	return postState(endpoint, map[string]any{keyOn: false})
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
