package profile

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	// The image may lack a zoneinfo database; quietHours.timezone must
	// resolve anyway.
	_ "time/tzdata"

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

// FxMap is the FALLBACK from effect names to WLED effect IDs. It is used only
// when a device's own list (GET /json/eff, see wled.ResolveEffect) cannot be
// read: WLED numbers effects by their position in that list, and the
// numbering moves between releases. The first version of this table was off
// for most names on WLED 16 -- "Twinkle" at 5 showed Random Colors (#77).
//
// IDs as measured on WLED 16.0.1 (ESP32, 220 effects). Candle is WLED's
// FX_MODE_CANDLE. A name missing here still works whenever the device answers.
var FxMap = map[string]int{
	"Solid":         0,
	"Blink":         1,
	"Breathe":       2,
	"Wipe":          3,
	"Dynamic":       7,
	"Rainbow":       9,
	"Scan":          10,
	"Twinkle":       17,
	"Strobe":        23,
	"Chase":         28,
	"Chase Rainbow": 30,
	"Aurora":        38,
	"Fireworks":     42,
	"Candle":        88,
	"DJ Light":      159,
	"Blurz":         163,
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
	Tags []string `yaml:"tags"`
	Fx   string   `yaml:"fx"`
	// Duration is whole seconds, or "auto": as long as the LED matrix at
	// Display shows something (#77).
	Duration Duration `yaml:"duration"`
	// Display is the led-catcher base URL a duration of "auto" follows
	// (GET {Display}/display). Unset: LED_CATCHER_URL.
	Display string `yaml:"display"`
	// Color is a local palette or single color (see GetColor). A name that is
	// neither is looked up as a device palette (GET /json/pal), so WLED's own
	// palettes such as "Lava" work too.
	Color string `yaml:"color"`
	// Palette names a device palette explicitly -- for one whose name a local
	// color shadows ("Sunset"), or to combine a palette with Color. A number is
	// used as the palette ID directly.
	Palette string `yaml:"palette"`
	// Segments are the WLED segment IDs the effect is sent to. Empty sends it
	// to the main segment, as before.
	Segments []int  `yaml:"segments"`
	Endpoint string `yaml:"endpoint"`

	// Speed (sx) and Intensity (ix), 0-255. Unset: 128 and 255.
	Speed     *int `yaml:"speed"`
	Intensity *int `yaml:"intensity"`
	// Brightness (bri), 0-255. Unset: the device keeps its brightness.
	Brightness *int `yaml:"brightness"`
	// Transition in seconds, sent as WLED's one-off tt (100 ms units). Unset:
	// the device's own transition.
	Transition *float64 `yaml:"transition"`
	// Restore puts the device back into the state it had before the effect
	// once Duration is over, instead of switching it off -- a device with an
	// ambient scene no longer goes dark after every event.
	Restore bool `yaml:"restore"`
}

// Validate checks the numeric fields against WLED's ranges.
func (e Effect) Validate() error {
	for name, v := range map[string]*int{"speed": e.Speed, "intensity": e.Intensity, "brightness": e.Brightness} {
		if v != nil && (*v < 0 || *v > 255) {
			return fmt.Errorf("%s %d out of range 0-255", name, *v)
		}
	}
	if e.Transition != nil && (*e.Transition < 0 || *e.Transition > 6553.5) {
		return fmt.Errorf("transition %.1fs out of range 0-6553.5", *e.Transition)
	}
	for _, s := range e.Segments {
		if s < 0 || s > 31 {
			return fmt.Errorf("segment %d out of range 0-31", s)
		}
	}
	return nil
}

// Duration is how long an effect runs before the device is switched off or
// restored.
type Duration struct {
	// Seconds is the fixed duration; 0 leaves the effect on.
	Seconds int
	// Auto ends the effect when the led-catcher's panel has finished showing
	// the message instead: the matrix scrolls a text for (64 + width) x 30 ms
	// and drops messages while it is busy, so a fixed duration drifts apart
	// from it in a burst.
	Auto bool
}

// UnmarshalYAML accepts a number of seconds or the string "auto".
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value.Tag == "!!str" && strings.EqualFold(strings.TrimSpace(value.Value), "auto") {
		*d = Duration{Auto: true}
		return nil
	}
	var seconds int
	if err := value.Decode(&seconds); err != nil {
		return fmt.Errorf("duration: want seconds or \"auto\", got %q", value.Value)
	}
	*d = Duration{Seconds: seconds}
	return nil
}

// String renders the duration as it is written in a profile.
func (d Duration) String() string {
	if d.Auto {
		return "auto"
	}
	return strconv.Itoa(d.Seconds)
}

// Configuration represents the top-level profile YAML.
type Configuration struct {
	Effects map[string]Effect `yaml:"effects"`

	// QuietHours, when set, lets only its Allow severities match (#82).
	QuietHours *QuietHours `yaml:"quietHours"`

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

// QuietHours is a window in which only the Allow severities reach the strip
// (#82): From to To, which may cross midnight, plus all of Saturday and Sunday
// with Weekends. Times are HH:MM in Timezone (an IANA name; empty is local).
// The same schema as led-catcher's quietHours, so one profile reads the same
// on both catchers.
type QuietHours struct {
	From     string   `yaml:"from"`
	To       string   `yaml:"to"`
	Weekends bool     `yaml:"weekends"`
	Timezone string   `yaml:"timezone"`
	Allow    []string `yaml:"allow"`
}

// now is the clock MatchEffect checks quiet hours against; tests replace it.
var now = time.Now

// Validate checks the times and the time zone.
func (q QuietHours) Validate() error {
	if _, err := time.Parse("15:04", q.From); err != nil {
		return fmt.Errorf("quietHours.from %q: want HH:MM", q.From)
	}
	if _, err := time.Parse("15:04", q.To); err != nil {
		return fmt.Errorf("quietHours.to %q: want HH:MM", q.To)
	}
	if _, err := time.LoadLocation(q.Timezone); err != nil {
		return fmt.Errorf("quietHours.timezone %q: %w", q.Timezone, err)
	}
	return nil
}

// Active reports whether t falls into the quiet window.
func (q QuietHours) Active(t time.Time) bool {
	if loc, err := time.LoadLocation(q.Timezone); err == nil {
		t = t.In(loc)
	}
	if q.Weekends && (t.Weekday() == time.Saturday || t.Weekday() == time.Sunday) {
		return true
	}
	from, errFrom := time.Parse("15:04", q.From)
	to, errTo := time.Parse("15:04", q.To)
	if errFrom != nil || errTo != nil {
		return false
	}
	minute := t.Hour()*60 + t.Minute()
	start := from.Hour()*60 + from.Minute()
	end := to.Hour()*60 + to.Minute()
	if start <= end {
		return minute >= start && minute < end
	}
	return minute >= start || minute < end
}

// Allows reports whether severity gets through during quiet hours. No Allow
// list means error and critical.
func (q QuietHours) Allows(severity string) bool {
	allow := q.Allow
	if len(allow) == 0 {
		allow = []string{"error", "critical"}
	}
	for _, a := range allow {
		if strings.EqualFold(a, severity) {
			return true
		}
	}
	return false
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
	if config.QuietHours != nil {
		if err := config.QuietHours.Validate(); err != nil {
			return Configuration{}, fmt.Errorf("invalid profile: %w", err)
		}
	}

	return config, nil
}

// MatchEffect finds the first effect matching the given system, severity and
// tags. Effects are evaluated in profile document order and the first match
// wins, so a specific rule must be declared above a wildcard rule it overlaps
// with. Systems support wildcard "*" to match any system. System and severity
// are both matched case-insensitively — homerun2 producers emit lowercase
// values while profiles are often authored in another case, and a rule that
// silently never fires over "Tabletennis" vs "tabletennis" helps nobody (#77).
// An effect with tags only matches if every one of them is present in the
// message's tags (see TagsMatch). During the profile's quiet hours, a
// severity it does not allow matches nothing.
func MatchEffect(config Configuration, system, severity, tags string) (Effect, bool) {
	if q := config.QuietHours; q != nil && !q.Allows(severity) && q.Active(now()) {
		return Effect{}, false
	}
	for _, name := range config.Names() {
		effect := config.Effects[name]
		systemMatch := false
		for _, s := range effect.Systems {
			if s == "*" || strings.EqualFold(s, system) {
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
