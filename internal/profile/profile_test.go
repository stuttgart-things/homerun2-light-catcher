package profile

import (
	"os"
	"path/filepath"
	"testing"
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

	effect, found := MatchEffect(config, "github", "ERROR")
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

	effect, found := MatchEffect(config, "some-random-system", "INFO")
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

	_, found := MatchEffect(config, "github", "UNKNOWN")
	if found {
		t.Error("expected no match for unknown severity")
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
