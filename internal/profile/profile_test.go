package profile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

const testProfileYAML = `---
effects:
  error-git:
    systems:
      - gitlab
      - github
    severity:
      - ERROR
    fx: Blurz
    duration: 3
    color: sunset
    segments:
      - 0
    endpoint: http://wled:8080
  info:
    systems:
      - "*"
    severity:
      - INFO
    fx: DJ Light
    duration: 3
    color: ocean
    segments:
      - 0
    endpoint: http://localhost:8080
  success:
    systems:
      - "*"
    severity:
      - SUCCESS
    fx: Aurora
    duration: 3
    color: forest
    segments:
      - 0
    endpoint: http://localhost:8080
`

func writeTestProfile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	if err := os.WriteFile(path, []byte(testProfileYAML), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfiguration(t *testing.T) {
	path := writeTestProfile(t)
	config, err := LoadConfiguration(path)
	if err != nil {
		t.Fatalf("LoadConfiguration: %v", err)
	}
	if len(config.Effects) != 3 {
		t.Errorf("expected 3 effects, got %d", len(config.Effects))
	}
	if e, ok := config.Effects["error-git"]; !ok {
		t.Error("missing error-git effect")
	} else if e.Fx != "Blurz" {
		t.Errorf("expected fx Blurz, got %s", e.Fx)
	}
}

func TestLoadConfiguration_MissingFile(t *testing.T) {
	_, err := LoadConfiguration("/nonexistent/profile.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadConfiguration_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte("{{invalid yaml"), 0644)
	_, err := LoadConfiguration(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestMatchEffect_ExactSystem(t *testing.T) {
	path := writeTestProfile(t)
	config, _ := LoadConfiguration(path)

	effect, found := MatchEffect(config, "github", "ERROR", "")
	if !found {
		t.Fatal("expected match for github/ERROR")
	}
	if effect.Fx != "Blurz" {
		t.Errorf("expected Blurz, got %s", effect.Fx)
	}
}

func TestMatchEffect_Wildcard(t *testing.T) {
	path := writeTestProfile(t)
	config, _ := LoadConfiguration(path)

	effect, found := MatchEffect(config, "some-random-system", "INFO", "")
	if !found {
		t.Fatal("expected wildcard match for INFO")
	}
	if effect.Fx != "DJ Light" {
		t.Errorf("expected DJ Light, got %s", effect.Fx)
	}
}

func TestMatchEffect_NoMatch(t *testing.T) {
	path := writeTestProfile(t)
	config, _ := LoadConfiguration(path)

	_, found := MatchEffect(config, "github", "UNKNOWN", "")
	if found {
		t.Error("expected no match for unknown severity")
	}
}

func TestMatchEffect_SeverityCaseInsensitive(t *testing.T) {
	path := writeTestProfile(t)
	config, _ := LoadConfiguration(path)

	// Profile authored as "INFO"; producers emit lowercase "info".
	effect, found := MatchEffect(config, "some-system", "info", "")
	if !found {
		t.Fatal("expected case-insensitive match for info → INFO")
	}
	if effect.Fx != "DJ Light" {
		t.Errorf("expected DJ Light, got %s", effect.Fx)
	}
}

func TestGetColor_Palette(t *testing.T) {
	colors, err := GetColor("sunset")
	if err != nil {
		t.Fatalf("GetColor sunset: %v", err)
	}
	if len(colors) != 6 {
		t.Errorf("sunset palette: expected 6 colors, got %d", len(colors))
	}
}

func TestGetColor_Single(t *testing.T) {
	colors, err := GetColor("red")
	if err != nil {
		t.Fatalf("GetColor red: %v", err)
	}
	if len(colors) != 1 {
		t.Errorf("expected 1 color, got %d", len(colors))
	}
	if colors[0] != [3]int{255, 0, 0} {
		t.Errorf("expected [255,0,0], got %v", colors[0])
	}
}

func TestGetColor_Unknown(t *testing.T) {
	_, err := GetColor("nonexistent")
	if err == nil {
		t.Error("expected error for unknown color")
	}
}

func TestFxMap(t *testing.T) {
	if FxMap["Solid"] != 0 {
		t.Errorf("expected Solid=0, got %d", FxMap["Solid"])
	}
	if FxMap["DJ Light"] != 14 {
		t.Errorf("expected DJ Light=14, got %d", FxMap["DJ Light"])
	}
}

func TestReverseFxMap(t *testing.T) {
	rev := ReverseFxMap()
	if rev[0] != "Solid" {
		t.Errorf("expected 0=Solid, got %s", rev[0])
	}
	if rev[14] != "DJ Light" {
		t.Errorf("expected 14=DJ Light, got %s", rev[14])
	}
}

const overlappingProfileYAML = `---
effects:
  tabletennis-win:
    systems: [tabletennis]
    severity: [success]
    fx: Fireworks
  success:
    systems: ["*"]
    severity: [success]
    fx: Aurora
  zz-late:
    systems: [tabletennis]
    severity: [success]
    fx: Solid
`

func TestMatchEffect_DocumentOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	if err := os.WriteFile(path, []byte(overlappingProfileYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Map iteration is randomized per range statement, so repeating the
	// load+match catches any path that still walks the map.
	for i := 0; i < 200; i++ {
		config, err := LoadConfiguration(path)
		if err != nil {
			t.Fatalf("LoadConfiguration: %v", err)
		}

		effect, found := MatchEffect(config, "tabletennis", "success", "")
		if !found || effect.Fx != "Fireworks" {
			t.Fatalf("iteration %d: expected the specific rule declared first (Fireworks), got %q", i, effect.Fx)
		}

		effect, found = MatchEffect(config, "gitlab", "success", "")
		if !found || effect.Fx != "Aurora" {
			t.Fatalf("iteration %d: expected wildcard rule (Aurora), got %q", i, effect.Fx)
		}
	}
}

func TestMatchEffect_WildcardDeclaredFirstWins(t *testing.T) {
	var config Configuration
	yamlDoc := `effects:
  success:
    systems: ["*"]
    severity: [success]
    fx: Aurora
  tabletennis-win:
    systems: [tabletennis]
    severity: [success]
    fx: Fireworks
`
	if err := yaml.Unmarshal([]byte(yamlDoc), &config); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 200; i++ {
		effect, _ := MatchEffect(config, "tabletennis", "success", "")
		if effect.Fx != "Aurora" {
			t.Fatalf("iteration %d: first match wins regardless of specificity, expected Aurora, got %q", i, effect.Fx)
		}
	}
}

func TestConfigurationNames(t *testing.T) {
	path := writeTestProfile(t)
	config, err := LoadConfiguration(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := config.Names(), []string{"error-git", "info", "success"}; !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want document order %v", got, want)
	}

	// A Configuration built in code has no document order; names are sorted.
	built := Configuration{Effects: map[string]Effect{"b": {}, "a": {}, "c": {}}}
	if got, want := built.Names(), []string{"a", "b", "c"}; !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want sorted %v", got, want)
	}
}

func TestTagsMatch(t *testing.T) {
	tests := []struct {
		name     string
		required []string
		tags     string
		want     bool
	}{
		{"no required tags, no message tags", nil, "", true},
		{"no required tags, message tags", nil, "side=a", true},
		{"single tag present", []string{"side=a"}, "match=1,set=2,side=a", true},
		{"substring is not a match", []string{"side=a"}, "side=ab", false},
		{"superstring is not a match", []string{"side=ab"}, "side=a", false},
		{"all tags required (AND)", []string{"transition=point", "side=a"}, "transition=point,side=b", false},
		{"all tags present in any order", []string{"side=a", "transition=point"}, "match=36c17b30,set=2,transition=point,side=a", true},
		{"whitespace around elements ignored", []string{" side=a "}, "set=2, side=a ,x", true},
		{"case-sensitive", []string{"side=A"}, "side=a", false},
		{"required tag, message without tags", []string{"side=a"}, "", false},
		{"empty required entry never matches", []string{""}, "a,,b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TagsMatch(tt.required, tt.tags); got != tt.want {
				t.Errorf("TagsMatch(%q, %q) = %v, want %v", tt.required, tt.tags, got, tt.want)
			}
		})
	}
}

const tagProfileYAML = `effects:
  tabletennis-match:
    systems: [tabletennis]
    severity: [success]
    tags: [transition=match_won]
    fx: Fireworks
  tabletennis-point-a:
    systems: [tabletennis]
    severity: [info]
    tags: [transition=point, side=a]
    fx: Solid
    color: blue
  tabletennis-point-b:
    systems: [tabletennis]
    severity: [info]
    tags: [transition=point, side=b]
    fx: Solid
    color: red
  success:
    systems: ["*"]
    severity: [success]
    fx: Aurora
  info:
    systems: ["*"]
    severity: [info]
    fx: DJ Light
`

func TestMatchEffect_Tags(t *testing.T) {
	var config Configuration
	if err := yaml.Unmarshal([]byte(tagProfileYAML), &config); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name              string
		system, sev, tags string
		wantFx, wantColor string
	}{
		{"tag rule precedes wildcard declared after it", "tabletennis", "SUCCESS", "match=36c17b30,set=3,transition=match_won,side=a", "Fireworks", ""},
		{"side a point", "tabletennis", "INFO", "match=36c17b30,set=2,transition=point,side=a", "Solid", "blue"},
		{"side b point", "tabletennis", "INFO", "match=36c17b30,set=2,transition=point,side=b", "Solid", "red"},
		{"partial tags fall through to wildcard", "tabletennis", "INFO", "transition=point", "DJ Light", ""},
		{"set won falls through to wildcard", "tabletennis", "SUCCESS", "transition=set_won,side=a", "Aurora", ""},
		{"no tags falls through to wildcard", "tabletennis", "INFO", "", "DJ Light", ""},
		{"tags on another system do not match tag rules", "other", "INFO", "transition=point,side=a", "DJ Light", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 50; i++ {
				effect, found := MatchEffect(config, tt.system, tt.sev, tt.tags)
				if !found {
					t.Fatal("expected a match")
				}
				if effect.Fx != tt.wantFx || effect.Color != tt.wantColor {
					t.Fatalf("got %s/%s, want %s/%s", effect.Fx, effect.Color, tt.wantFx, tt.wantColor)
				}
			}
		})
	}
}

func TestMatchEffect_ProfileWithoutTagsIgnoresMessageTags(t *testing.T) {
	path := writeTestProfile(t)
	config, err := LoadConfiguration(path)
	if err != nil {
		t.Fatal(err)
	}

	effect, found := MatchEffect(config, "github", "ERROR", "transition=point,side=a")
	if !found || effect.Fx != "Blurz" {
		t.Errorf("expected Blurz regardless of message tags, got %q (found=%v)", effect.Fx, found)
	}
}
