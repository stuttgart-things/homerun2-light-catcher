package mock

import (
	"encoding/json"
	"fmt"
	"os"
)

// wled16Effects is what the mock answers on GET /json/eff: a JSON array whose
// index is the effect ID, as on a real device.
//
// It reproduces WLED 16.0.1 where the numbering is known -- IDs 0-42 in order,
// plus Candle, DJ Light and Blurz at their device positions -- and marks every
// other slot "RSVD", which is what WLED itself returns for an unavailable
// effect. So the light-catcher's lookup resolves the same IDs against the
// mock as against the device (#77). Before, the mock accepted any ID, which
// is why the wrong table never showed up anywhere but on real hardware.
//
// For a full list, capture it from a device and point WLED_EFFECTS_FILE at it:
//
//	curl http://<wled>/json/eff > effects.json
var wled16Effects = func() []string {
	known := []string{
		"Solid", "Blink", "Breathe", "Wipe", "Wipe Random", "Random Colors",
		"Sweep", "Dynamic", "Colorloop", "Rainbow", "Scan", "Scan Dual", "Fade",
		"Theater", "Theater Rainbow", "Running", "Saw", "Twinkle", "Dissolve",
		"Dissolve Rnd", "Sparkle", "Sparkle Dark", "Sparkle+", "Strobe",
		"Strobe Rainbow", "Strobe Mega", "Blink Rainbow", "Android", "Chase",
		"Chase Random", "Chase Rainbow", "Chase Flash", "Chase Flash Rnd",
		"Rainbow Runner", "Colorful", "Traffic Light", "Sweep Random", "Chase 2",
		"Aurora", "Stream", "Scanner", "Lighthouse", "Fireworks",
	}
	sparse := map[int]string{
		88:  "Candle",
		159: "DJ Light",
		163: "Blurz",
	}
	list := make([]string, 164)
	for i := range list {
		list[i] = "RSVD"
	}
	copy(list, known)
	for id, name := range sparse {
		list[id] = name
	}
	return list
}()

// loadEffects returns the effect list the mock serves: the file named by
// WLED_EFFECTS_FILE if set, else the built-in WLED 16 list.
func loadEffects() ([]string, error) {
	path := os.Getenv("WLED_EFFECTS_FILE")
	if path == "" {
		return wled16Effects, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read WLED_EFFECTS_FILE: %w", err)
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return nil, fmt.Errorf("parse WLED_EFFECTS_FILE %s: %w", path, err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("WLED_EFFECTS_FILE %s holds no effects", path)
	}
	return names, nil
}

// effectNamesByID maps IDs to display names, leaving out reserved slots.
func effectNamesByID(effects []string) map[int]string {
	names := make(map[int]string, len(effects))
	for id, n := range effects {
		if n != "" && n != "RSVD" && n != "-" {
			names[id] = n
		}
	}
	return names
}
