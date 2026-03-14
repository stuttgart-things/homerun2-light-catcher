package profile

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Predefined color palettes.
var colorPalettes = map[string][][]int{
	"sunset": {
		{255, 94, 77},
		{255, 129, 78},
		{255, 0, 0},
		{255, 178, 97},
		{255, 225, 130},
		{255, 0, 0},
	},
	"beach": {
		{241, 213, 145},
		{118, 207, 233},
		{37, 110, 146},
		{235, 223, 142},
	},
	"forest": {
		{34, 139, 34},
		{0, 128, 0},
		{85, 107, 47},
		{34, 139, 34},
		{34, 139, 34},
		{85, 107, 47},
	},
	"ocean": {
		{0, 105, 148},
		{70, 130, 180},
		{135, 206, 250},
		{240, 248, 255},
	},
}

// Single color names.
var singleColors = map[string][3]int{
	"red":    {255, 0, 0},
	"yellow": {255, 255, 0},
	"green":  {0, 255, 0},
	"blue":   {0, 0, 255},
	"white":  {255, 255, 255},
}

// FxMap maps WLED effect names to effect IDs.
var FxMap = map[string]int{
	"Solid":         0,
	"Blink":         1,
	"Breathe":       2,
	"Wipe":          3,
	"Scan":          4,
	"Twinkle":       5,
	"Fireworks":     6,
	"Rainbow":       7,
	"Candle":        8,
	"Chase":         9,
	"Dynamic":       10,
	"Chase Rainbow": 11,
	"Aurora":        12,
	"Blurz":         13,
	"DJ Light":      14,
}

// ReverseFxMap returns a mapping of effect IDs to names.
func ReverseFxMap() map[int]string {
	reverse := make(map[int]string, len(FxMap))
	for name, id := range FxMap {
		reverse[id] = name
	}
	return reverse
}

// Effect represents a single effect entry in the profile.
type Effect struct {
	Systems  []string `yaml:"systems"`
	Severity []string `yaml:"severity"`
	Fx       string   `yaml:"fx"`
	Duration int      `yaml:"duration"`
	Color    string   `yaml:"color"`
	Segments []int    `yaml:"segments"`
	Endpoint string   `yaml:"endpoint"`
}

// Configuration represents the top-level profile YAML.
type Configuration struct {
	Effects map[string]Effect `yaml:"effects"`
}

// LoadConfiguration reads and parses a profile YAML file.
func LoadConfiguration(filepath string) (Configuration, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return Configuration{}, fmt.Errorf("failed to read profile: %w", err)
	}

	var config Configuration
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Configuration{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return config, nil
}

// MatchEffect finds the first effect matching the given system and severity.
// Systems support wildcard "*" to match any system.
func MatchEffect(config Configuration, system, severity string) (Effect, bool) {
	for _, effect := range config.Effects {
		systemMatch := false
		for _, s := range effect.Systems {
			if s == "*" || s == system {
				systemMatch = true
				break
			}
		}

		severityMatch := false
		for _, sev := range effect.Severity {
			if sev == severity {
				severityMatch = true
				break
			}
		}

		if systemMatch && severityMatch {
			return effect, true
		}
	}
	return Effect{}, false
}

// GetColor resolves a color name to RGB values. Supports palette names and single colors.
func GetColor(name string) ([][3]int, error) {
	if palette, ok := colorPalettes[name]; ok {
		result := make([][3]int, len(palette))
		for i, c := range palette {
			result[i] = [3]int{c[0], c[1], c[2]}
		}
		return result, nil
	}

	if color, ok := singleColors[name]; ok {
		return [][3]int{color}, nil
	}

	return nil, fmt.Errorf("unknown color or palette: %s", name)
}
