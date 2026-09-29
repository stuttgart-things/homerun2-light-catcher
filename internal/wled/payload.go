package wled

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/profile"
)

// Defaults for a segment when the profile does not say otherwise; they are
// what every effect was sent with before speed and intensity were settable.
const (
	defaultSpeed     = 128
	defaultIntensity = 255
)

// WLED state keys used in more than one place.
const (
	keyOn  = "on"
	keyBri = "bri"
)

// effectPayload is one effect as it goes to POST /json/state.
type effectPayload struct {
	fx int
	// colors are sent as the segment's col; nil leaves the device's colors.
	colors [][3]int
	// palette, if set, is sent as the segment's pal.
	palette *int
	// segments are the segment IDs the effect goes to; empty sends one
	// segment without an id, which WLED applies to the main segment.
	segments  []int
	speed     int
	intensity int
	// brightness, if set, is the master brightness (bri).
	brightness *int
	// transition, if set, is WLED's one-off tt in 100 ms units.
	transition *int
	meta       EffectMeta
}

// payloadFor builds the payload for a profile effect whose fx, colors and
// palette are already resolved.
func payloadFor(effect profile.Effect, fx int, colors [][3]int, palette *int, meta EffectMeta) effectPayload {
	p := effectPayload{
		fx:         fx,
		colors:     colors,
		palette:    palette,
		segments:   effect.Segments,
		speed:      defaultSpeed,
		intensity:  defaultIntensity,
		brightness: effect.Brightness,
		meta:       meta,
	}
	if effect.Speed != nil {
		p.speed = *effect.Speed
	}
	if effect.Intensity != nil {
		p.intensity = *effect.Intensity
	}
	if effect.Transition != nil {
		tt := int(math.Round(*effect.Transition * 10))
		p.transition = &tt
	}
	return p
}

// body renders the payload as WLED's JSON state. The underscore fields are
// ignored by a real device; the mock shows them in its timeline.
func (p effectPayload) body() map[string]any {
	seg := func(id *int) map[string]any {
		s := map[string]any{"fx": p.fx, "sx": p.speed, "ix": p.intensity}
		if id != nil {
			s["id"] = *id
		}
		if p.colors != nil {
			s["col"] = p.colors
		}
		if p.palette != nil {
			s["pal"] = *p.palette
		}
		return s
	}

	var segs []map[string]any
	if len(p.segments) == 0 {
		segs = []map[string]any{seg(nil)}
	}
	for _, id := range p.segments {
		segs = append(segs, seg(&id))
	}

	b := map[string]any{
		keyOn:       true,
		"seg":       segs,
		"_severity": p.meta.Severity,
		"_system":   p.meta.System,
		"_effect":   p.meta.Effect,
		"_color":    p.meta.Color,
	}
	if p.brightness != nil {
		b[keyBri] = *p.brightness
	}
	if p.transition != nil {
		b["tt"] = *p.transition
	}
	if len(p.meta.Tags) > 0 {
		b["_tags"] = p.meta.Tags
	}
	return b
}

// restorableSegmentKeys are the segment fields worth writing back. Bounds
// (start/stop/len) and read-only fields stay as they are on the device.
var restorableSegmentKeys = []string{"id", keyOn, keyBri, "fx", "sx", "ix", "pal", "col"}

// fetchRestoreState reads GET {endpoint}/json/state and keeps what restoring
// needs: on, bri and each segment's look. It is what a device showed before
// the light-catcher touched it -- an ambient scene, typically.
func fetchRestoreState(endpoint string) (map[string]any, error) {
	resp, err := effectLookupClient.Get(endpoint + "/json/state")
	if err != nil {
		return nil, fmt.Errorf("GET %s/json/state: %w", endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s/json/state: %s", endpoint, resp.Status)
	}
	var state map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("decode %s/json/state: %w", endpoint, err)
	}

	out := map[string]any{}
	for _, k := range []string{keyOn, keyBri} {
		if v, ok := state[k]; ok {
			out[k] = v
		}
	}
	if segs, ok := state["seg"].([]any); ok {
		var keep []map[string]any
		for i, raw := range segs {
			s, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			k := map[string]any{}
			for _, key := range restorableSegmentKeys {
				if v, ok := s[key]; ok {
					k[key] = v
				}
			}
			// Without an id WLED applies segments by position; pin it so the
			// restore lands where it was read from.
			if _, ok := k["id"]; !ok {
				k["id"] = i
			}
			keep = append(keep, k)
		}
		if keep != nil {
			out["seg"] = keep
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s/json/state holds nothing to restore", endpoint)
	}
	return out, nil
}
