package wled

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/mock"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/profile"
)

func intp(v int) *int           { return &v }
func floatp(v float64) *float64 { return &v }

func mockServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(mock.NewServer("test", "abc1234", "2026-01-01").Handler())
	t.Cleanup(srv.Close)
	return srv
}

func writeProfile(t *testing.T, yaml string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func postRaw(t *testing.T, url, body string) {
	t.Helper()
	resp, err := http.Post(url+"/json/state", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s: %s", body, resp.Status)
	}
}

func shortDurations(t *testing.T) {
	t.Helper()
	orig := durationUnit
	durationUnit = 50 * time.Millisecond
	t.Cleanup(func() { durationUnit = orig })
}

// Segments, speed, intensity, brightness and transition from the profile reach
// the payload; unset ones keep the old defaults or stay off the wire (#77).
func TestPayload_ProfileFields(t *testing.T) {
	e := profile.Effect{
		Segments: []int{0, 2}, Speed: intp(40), Intensity: intp(200),
		Brightness: intp(90), Transition: floatp(1.5),
	}
	b := payloadFor(e, 42, [][3]int{{1, 2, 3}}, nil, EffectMeta{}).body()

	segs := b["seg"].([]map[string]any)
	if len(segs) != 2 || segs[0]["id"] != 0 || segs[1]["id"] != 2 {
		t.Fatalf("want segments 0 and 2 with ids, got %v", segs)
	}
	if segs[1]["sx"] != 40 || segs[1]["ix"] != 200 || segs[1]["fx"] != 42 {
		t.Fatalf("speed/intensity/fx not applied: %v", segs[1])
	}
	if b[keyBri] != 90 || b["tt"] != 15 {
		t.Fatalf("want bri 90 and tt 15 (1.5 s), got bri %v tt %v", b["bri"], b["tt"])
	}

	plain := payloadFor(profile.Effect{}, 0, [][3]int{{1, 2, 3}}, nil, EffectMeta{}).body()
	seg := plain["seg"].([]map[string]any)[0]
	if _, ok := seg["id"]; ok {
		t.Error("no segments in the profile: no id, the device's main segment")
	}
	if seg["sx"] != defaultSpeed || seg["ix"] != defaultIntensity {
		t.Errorf("defaults changed: %v", seg)
	}
	for _, k := range []string{keyBri, "tt"} {
		if _, ok := plain[k]; ok {
			t.Errorf("%s must stay off the wire when unset", k)
		}
	}
}

// A color that is not local is a device palette; a local color stays local,
// even where the device has a palette of the same name.
func TestResolveLook(t *testing.T) {
	srv := mockServer(t)

	colors, pal, err := resolveLook(profile.Effect{Color: "Lava", Endpoint: srv.URL})
	if err != nil || colors != nil || pal == nil || *pal != 8 {
		t.Fatalf("Lava must be device palette 8, got colors %v pal %v err %v", colors, pal, err)
	}

	colors, pal, err = resolveLook(profile.Effect{Color: "sunset", Endpoint: srv.URL})
	if err != nil || colors == nil || pal != nil {
		t.Fatalf("sunset is a local palette and wins: colors %v pal %v err %v", colors, pal, err)
	}

	colors, pal, err = resolveLook(profile.Effect{Color: "red", Palette: "Sunset", Endpoint: srv.URL})
	if err != nil || colors == nil || pal == nil || *pal != 13 {
		t.Fatalf("explicit palette Sunset (13) alongside red: colors %v pal %v err %v", colors, pal, err)
	}

	if _, _, err := resolveLook(profile.Effect{Color: "no-such", Palette: "Lava", Endpoint: srv.URL}); err == nil {
		t.Error("an unknown color next to a palette is a typo, not a palette")
	}
	if _, _, err := resolveLook(profile.Effect{Color: "no-such", Endpoint: srv.URL}); err == nil {
		t.Error("neither a local color nor a device palette must fail")
	}
}

func TestValidate(t *testing.T) {
	for name, e := range map[string]profile.Effect{
		"speed":      {Speed: intp(256)},
		"brightness": {Brightness: intp(-1)},
		"transition": {Transition: floatp(-0.1)},
		"segment":    {Segments: []int{32}},
	} {
		if e.Validate() == nil {
			t.Errorf("%s out of range must be refused", name)
		}
	}
	if err := (profile.Effect{Speed: intp(0), Segments: []int{0, 31}}).Validate(); err != nil {
		t.Errorf("0 and 31 are valid: %v", err)
	}
}

// The #77 case: a device with an ambient scene no longer goes dark after an
// event when the effect says restore.
func TestSendToWLED_RestoreBringsBackTheScene(t *testing.T) {
	shortDurations(t)
	srv := mockServer(t)
	postRaw(t, srv.URL, `{"on":true,"bri":77,"seg":[{"fx":9,"pal":11,"sx":30,"ix":40,"col":[[1,2,3]]}]}`)

	path := writeProfile(t, `effects:
  alert:
    systems: ["*"]
    severity: [error]
    fx: Fireworks
    duration: 2
    color: red
    restore: true
    endpoint: `+srv.URL+"\n")

	SendToWLED(path, "error", "ci", "", nil)
	if st := mockState(t, srv.URL); st.Seg[0].Fx != 42 {
		t.Fatalf("the effect must show first, got fx %d", st.Seg[0].Fx)
	}
	time.Sleep(250 * time.Millisecond)

	st := mockState(t, srv.URL)
	if !st.On || st.Bri != 77 || st.Seg[0].Fx != 9 || st.Seg[0].Pal != 11 || st.Seg[0].Col[0] != [3]int{1, 2, 3} {
		t.Fatalf("the ambient scene must be back, got %+v", st)
	}
}

// Two effects in quick succession: the second must not capture the first as
// its "before" state -- the scene from before both comes back.
func TestSendToWLED_RestoreSkipsOurOwnEffect(t *testing.T) {
	shortDurations(t)
	srv := mockServer(t)
	postRaw(t, srv.URL, `{"on":true,"seg":[{"fx":9,"pal":11}]}`)

	path := writeProfile(t, `effects:
  first:
    systems: [a]
    severity: [info]
    fx: Twinkle
    duration: 1
    color: blue
    restore: true
    endpoint: `+srv.URL+`
  second:
    systems: [b]
    severity: [info]
    fx: Fireworks
    duration: 3
    color: red
    restore: true
    endpoint: `+srv.URL+"\n")

	SendToWLED(path, "info", "a", "", nil)
	SendToWLED(path, "info", "b", "", nil)
	time.Sleep(300 * time.Millisecond)

	if st := mockState(t, srv.URL); st.Seg[0].Fx != 9 || st.Seg[0].Pal != 11 {
		t.Fatalf("want the ambient scene (fx 9, pal 11), got fx %d pal %d", st.Seg[0].Fx, st.Seg[0].Pal)
	}
}

// Without restore the device goes off as before.
func TestSendToWLED_NoRestoreSwitchesOff(t *testing.T) {
	shortDurations(t)
	srv := mockServer(t)
	postRaw(t, srv.URL, `{"on":true,"seg":[{"fx":9}]}`)

	path := writeProfile(t, `effects:
  alert:
    systems: ["*"]
    severity: [error]
    fx: Solid
    duration: 1
    color: red
    endpoint: `+srv.URL+"\n")

	SendToWLED(path, "error", "ci", "", nil)
	time.Sleep(200 * time.Millisecond)
	if mockState(t, srv.URL).On {
		t.Fatal("without restore the device must be off")
	}
}

// Segments reach the mock as addressed segments; a palette-only update keeps
// the segment's colors, like WLED.
func TestSendToWLED_SegmentsAndPalette(t *testing.T) {
	srv := mockServer(t)
	path := writeProfile(t, `effects:
  lava:
    systems: ["*"]
    severity: [info]
    fx: Aurora
    color: Lava
    segments: [1]
    speed: 60
    endpoint: `+srv.URL+"\n")

	SendToWLED(path, "info", "x", "", nil)
	st := mockState(t, srv.URL)
	if len(st.Seg) != 2 {
		t.Fatalf("segment 1 must exist next to 0, got %d segments", len(st.Seg))
	}
	if s := st.Seg[1]; s.Fx != 38 || s.Pal != 8 || s.Sx != 60 {
		t.Fatalf("segment 1: want Aurora (38), Lava (8), sx 60, got %+v", s)
	}
	if st.Seg[0].Fx != 0 {
		t.Fatalf("segment 0 must be untouched, got fx %d", st.Seg[0].Fx)
	}
}
