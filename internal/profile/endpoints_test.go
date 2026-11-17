package profile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestEffectTargets(t *testing.T) {
	for name, tc := range map[string]struct {
		effect Effect
		want   []string
	}{
		"endpoint only":  {Effect{Endpoint: "http://strip"}, []string{"http://strip"}},
		"endpoints only": {Effect{Endpoints: []string{"http://strip", "http://mock"}}, []string{"http://strip", "http://mock"}},
		"both":           {Effect{Endpoint: "http://strip", Endpoints: []string{"http://mock"}}, []string{"http://strip", "http://mock"}},
		"duplicates":     {Effect{Endpoint: "http://strip", Endpoints: []string{"http://mock", " http://strip ", "http://mock"}}, []string{"http://strip", "http://mock"}},
		"empty":          {Effect{Endpoints: []string{"", "  "}}, []string{}},
	} {
		if got := tc.effect.Targets(); !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %v, want %v", name, got, tc.want)
		}
	}
}

func TestEffectValidate_Endpoints(t *testing.T) {
	if err := (Effect{}).Validate(); err == nil {
		t.Error("neither endpoint nor endpoints must be refused")
	}
	if err := (Effect{Endpoints: []string{""}}).Validate(); err == nil {
		t.Error("only empty endpoints must be refused")
	}
	if err := (Effect{Endpoints: []string{"http://mock"}}).Validate(); err != nil {
		t.Errorf("endpoints alone is valid: %v", err)
	}
	if err := (Effect{Endpoint: "http://strip"}).Validate(); err != nil {
		t.Errorf("endpoint alone is valid: %v", err)
	}
}

func TestLoadConfiguration_Endpoints(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.yaml")
	yaml := `effects:
  alert:
    systems: ["*"]
    severity: [error]
    fx: Blink
    color: red
    endpoint: http://strip
    endpoints:
      - http://mock
      - http://strip
  legacy:
    systems: ["*"]
    severity: [info]
    fx: Blink
    color: blue
    endpoint: http://strip
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := LoadConfiguration(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Effects["alert"].Targets(); !slices.Equal(got, []string{"http://strip", "http://mock"}) {
		t.Errorf("alert targets: %v", got)
	}
	if got := config.Effects["legacy"].Targets(); !slices.Equal(got, []string{"http://strip"}) {
		t.Errorf("legacy targets: %v", got)
	}
}
