package profile

import (
	"fmt"
	"os"
	"slices"
	"strings"

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
	// Tags, if set, must all be present in the message's comma-separated tags.
	Tags     []string `yaml:"tags"`
	Fx       string   `yaml:"fx"`
	Duration int      `yaml:"duration"`
	Color    string   `yaml:"color"`
	Segments []int    `yaml:"segments"`
	Endpoint string   `yaml:"endpoint"`
}

// Configuration represents the top-level profile YAML.
type Configuration struct {
	Effects map[string]Effect `yaml:"effects"`

	// order holds the effect names in profile document order.
	order []string
}

// UnmarshalYAML decodes the profile and records the document order of the
// effects mapping, which a Go map alone would lose.
func (c *Configuration) UnmarshalYAML(value *yaml.Node) error {
	type plain Configuration
	if err := value.Decode((*plain)(c)); err != nil {
		return err
	}

	c.order = nil
	if value.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(value.Content); i += 2 {
		if value.Content[i].Value != "effects" || value.Content[i+1].Kind != yaml.MappingNode {
			continue
		}
		effects := value.Content[i+1].Content
		for j := 0; j+1 < len(effects); j += 2 {
			c.order = append(c.order, effects[j].Value)
		}
	}
	return nil
}

// Names returns the effect names in profile document order. Effects without a
// recorded position (e.g. a Configuration built in code) follow, sorted by name.
func (c Configuration) Names() []string {
	names := make([]string, 0, len(c.Effects))
	seen := make(map[string]bool, len(c.Effects))
	for _, name := range c.order {
		if _, ok := c.Effects[name]; ok && !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}

	var rest []string
	for name := range c.Effects {
		if !seen[name] {
			rest = append(rest, name)
		}
	}
	slices.Sort(rest)
	return append(names, rest...)
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

// MatchEffect finds the first effect matching the given system, severity and
// tags. Effects are evaluated in profile document order and the first match
// wins, so a specific rule must be declared above a wildcard rule it overlaps
// with. Systems support wildcard "*" to match any system. Severity is matched
// case-insensitively — homerun2 producers emit lowercase severities while
// profiles are often authored in uppercase. An effect with tags only matches
// if every one of them is present in the message's tags (see TagsMatch).
func MatchEffect(config Configuration, system, severity, tags string) (Effect, bool) {
	for _, name := range config.Names() {
		effect := config.Effects[name]
		systemMatch := false
		for _, s := range effect.Systems {
			if s == "*" || s == system {
				systemMatch = true
				break
			}
		}

		severityMatch := false
		for _, sev := range effect.Severity {
			if strings.EqualFold(sev, severity) {
				severityMatch = true
				break
			}
		}

		if systemMatch && severityMatch && TagsMatch(effect.Tags, tags) {
			return effect, true
		}
	}
	return Effect{}, false
}

// TagsMatch reports whether every required tag equals one whole element of the
// comma-separated message tags: "side=a" matches "set=2,side=a" but not
// "side=ab". Surrounding whitespace is ignored; comparison is case-sensitive.
// No required tags always matches.
func TagsMatch(required []string, messageTags string) bool {
	if len(required) == 0 {
		return true
	}

	present := make(map[string]bool)
	for _, tag := range strings.Split(messageTags, ",") {
		if tag = strings.TrimSpace(tag); tag != "" {
			present[tag] = true
		}
	}

	for _, tag := range required {
		if !present[strings.TrimSpace(tag)] {
			return false
		}
	}
	return true
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
