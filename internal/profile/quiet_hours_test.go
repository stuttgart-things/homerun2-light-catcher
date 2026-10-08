package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Tests for #82: quiet hours let only a severity floor through.

const quietProfileYAML = `
quietHours:
  from: "19:00"
  to: "07:00"
  weekends: true
  timezone: Europe/Berlin
  allow: [error, critical]
effects:
  all:
    systems: ["*"]
    severity: [info, success, warning, error, critical]
    fx: Solid
    color: blue
`

func loadQuiet(t *testing.T, yaml string) Configuration {
	t.Helper()
	path := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfiguration(path)
	if err != nil {
		t.Fatalf("LoadConfiguration: %v", err)
	}
	return cfg
}

func at(t *testing.T, s string) {
	t.Helper()
	berlin, _ := time.LoadLocation("Europe/Berlin")
	ts, err := time.ParseInLocation("2006-01-02 15:04", s, berlin)
	if err != nil {
		t.Fatal(err)
	}
	old := now
	now = func() time.Time { return ts }
	t.Cleanup(func() { now = old })
}

func TestQuietHoursHoldBackLowSeverities(t *testing.T) {
	cfg := loadQuiet(t, quietProfileYAML)

	// Thursday evening, Friday early morning, Saturday noon.
	for _, ts := range []string{"2026-10-08 22:30", "2026-10-09 06:59", "2026-10-10 12:00"} {
		at(t, ts)
		if _, ok := MatchEffect(cfg, "homerun2-git-pitcher", "success", ""); ok {
			t.Errorf("%s: success must not match during quiet hours", ts)
		}
		if _, ok := MatchEffect(cfg, "msteams", "CRITICAL", ""); !ok {
			t.Errorf("%s: critical must match during quiet hours", ts)
		}
	}
}

func TestQuietHoursOfficeHoursShowEverything(t *testing.T) {
	cfg := loadQuiet(t, quietProfileYAML)
	at(t, "2026-10-08 07:00")

	if _, ok := MatchEffect(cfg, "homerun2-git-pitcher", "info", ""); !ok {
		t.Error("info must match during office hours")
	}
}

func TestQuietHoursUseTheirTimezone(t *testing.T) {
	q := QuietHours{From: "19:00", To: "07:00", Timezone: "Europe/Berlin"}

	// 17:30 UTC is 19:30 in Berlin (CEST).
	if !q.Active(time.Date(2026, 10, 8, 17, 30, 0, 0, time.UTC)) {
		t.Error("17:30 UTC must be quiet in Berlin")
	}
}

func TestQuietHoursWithinADay(t *testing.T) {
	q := QuietHours{From: "12:00", To: "13:00"}
	loc := time.Local

	if !q.Active(time.Date(2026, 10, 8, 12, 30, 0, 0, loc)) {
		t.Error("12:30 must be quiet")
	}
	if q.Active(time.Date(2026, 10, 8, 13, 0, 0, 0, loc)) {
		t.Error("13:00 must not be quiet")
	}
}

func TestQuietHoursDefaultAllow(t *testing.T) {
	q := QuietHours{}
	if !q.Allows("error") || !q.Allows("Critical") || q.Allows("warning") {
		t.Error("default allow must be error and critical")
	}
}

func TestQuietHoursValidation(t *testing.T) {
	for name, yaml := range map[string]string{
		"bad from":     "quietHours: {from: '7pm', to: '07:00'}\neffects: {}\n",
		"bad to":       "quietHours: {from: '19:00', to: '25:00'}\neffects: {}\n",
		"bad timezone": "quietHours: {from: '19:00', to: '07:00', timezone: Mars/Base}\neffects: {}\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "profile.yaml")
			if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadConfiguration(path); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestNoQuietHoursWithoutConfig(t *testing.T) {
	cfg := loadQuiet(t, "effects:\n  all:\n    systems: ['*']\n    severity: [info]\n    fx: Solid\n")
	at(t, "2026-10-10 23:00")

	if _, ok := MatchEffect(cfg, "x", "info", ""); !ok {
		t.Error("without quietHours everything matches")
	}
}

func TestGetColorOrange(t *testing.T) {
	colors, err := GetColor("orange")
	if err != nil || len(colors) != 1 || colors[0] != [3]int{255, 120, 0} {
		t.Errorf("orange: got %v, %v", colors, err)
	}
}
