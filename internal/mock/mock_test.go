package mock

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestHandleState_POST(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	body := `{"on":true,"bri":200,"seg":[{"fx":13,"sx":128,"ix":255,"col":[[255,0,0]]}]}`
	req := httptest.NewRequest(http.MethodPost, "/json/state", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var state WLEDState
	json.NewDecoder(w.Body).Decode(&state)

	if !state.On {
		t.Error("expected on=true")
	}
	if len(state.Seg) != 1 || state.Seg[0].Fx != 13 {
		t.Error("expected 1 segment with fx=13")
	}
}

func TestHandleState_GET(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/json/state", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var state WLEDState
	json.NewDecoder(w.Body).Decode(&state)
	if state.On {
		t.Error("expected initial state to be off")
	}
}

func TestHandleState_InvalidJSON(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	req := httptest.NewRequest(http.MethodPost, "/json/state", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleState_ValidationError(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	body := `{"on":true,"bri":300}`
	req := httptest.NewRequest(http.MethodPost, "/json/state", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bri=300, got %d", w.Code)
	}
}

func TestHandleAPIReset(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	// Set some state first
	body := `{"on":true,"bri":200,"seg":[{"fx":7,"sx":100,"ix":200,"col":[[0,255,0]]}]}`
	req := httptest.NewRequest(http.MethodPost, "/json/state", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Reset
	req = httptest.NewRequest(http.MethodPost, "/api/reset", http.NoBody)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify reset state
	req = httptest.NewRequest(http.MethodGet, "/json/state", http.NoBody)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var state WLEDState
	json.NewDecoder(w.Body).Decode(&state)
	if state.On {
		t.Error("expected state to be off after reset")
	}
}

func TestHandleHealthz(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	b, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(b), `"status":"ok"`) {
		t.Errorf("unexpected body: %s", string(b))
	}
}

func TestHandleAPIState(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/state", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp apiStateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.RequestCount != 0 {
		t.Errorf("expected 0 requests, got %d", resp.RequestCount)
	}
}

func TestValidateSegment_OutOfRange(t *testing.T) {
	seg := Segment{Fx: 256, Sx: 0, Ix: 0, Col: nil}
	if err := validateSegment(0, seg); err == nil {
		t.Error("expected error for fx=256")
	}
}

func TestValidateWLEDState_TooManySegments(t *testing.T) {
	segs := make([]Segment, 33)
	state := WLEDState{On: true, Bri: 128, Seg: segs}
	if err := validateWLEDState(state); err == nil {
		t.Error("expected error for 33 segments")
	}
}

// The mock answers /json/eff like WLED 16, so the light-catcher resolves the
// same IDs against it as against a device (#77).
func TestHandleEffects_WLED16Numbering(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/json/eff", http.NoBody))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var names []string
	if err := json.NewDecoder(w.Body).Decode(&names); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]string{0: fxSolid, 9: nameRainbow, 17: "Twinkle", 23: "Strobe", 38: "Aurora", 42: fxFireworks, 88: "Candle", 159: "DJ Light", 163: "Blurz", 43: reservedSlot} {
		if id >= len(names) || names[id] != want {
			t.Errorf("effect %d: want %q", id, want)
		}
	}
	if s.effectNames[42] != fxFireworks {
		t.Errorf("dashboard names must follow the same list, got %q", s.effectNames[42])
	}
	if _, ok := s.effectNames[43]; ok {
		t.Error("reserved slots are not names")
	}
}

func TestLoadEffects_FromFile(t *testing.T) {
	path := t.TempDir() + "/eff.json"
	if err := os.WriteFile(path, []byte(`["Solid","Custom"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WLED_EFFECTS_FILE", path)
	s := NewServer("test", "abc1234", "2026-01-01")
	if len(s.effects) != 2 || s.effectNames[1] != "Custom" {
		t.Fatalf("got %v", s.effects)
	}
	t.Setenv("WLED_EFFECTS_FILE", path+".missing")
	if s := NewServer("test", "abc1234", "2026-01-01"); len(s.effects) != len(wled16Effects) {
		t.Fatal("an unreadable file falls back to the built-in list")
	}
}

// Behind homerun2-portal the mock runs under BASE_PATH; the WLED API and the
// health endpoints stay at the root too.
func TestHandlerBasePath(t *testing.T) {
	const (
		get, post = http.MethodGet, http.MethodPost
		wled      = "/wled"
		ui        = wled + "/"
	)
	s := NewServer("test", "abc1234", "2026-01-01")
	s.SetBasePath(wled)
	handler := s.Handler()

	cases := []struct {
		method, path string
		code         int
		location     string
	}{
		{get, ui, 200, ""},
		{get, "/wled/json/state/events", 200, ""},
		{get, "/wled/api/state", 200, ""},
		{post, "/wled/api/reset", 200, ""},
		{get, "/json/state", 200, ""},
		{get, "/json/eff", 200, ""},
		{get, "/json/pal", 200, ""},
		{get, "/health", 200, ""},
		{get, "/healthz", 200, ""},
		{get, wled, 307, ui},
		{get, "/", 307, ui},
		{get, "/api/state", 404, ""},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, http.NoBody))
		if rec.Code != tc.code {
			t.Errorf("%s %s: code %d, want %d", tc.method, tc.path, rec.Code, tc.code)
			continue
		}
		if loc := rec.Header().Get("Location"); tc.location != "" && loc != tc.location {
			t.Errorf("%s %s: Location %q, want %q", tc.method, tc.path, loc, tc.location)
		}
	}
}

// Every URL the dashboard hands out must carry the base path: the portal
// forwards only /wled/... to the mock.
func TestDashboardCarriesBasePath(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	s.SetBasePath("/wled")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wled/", http.NoBody))
	page := rec.Body.String()

	if !strings.Contains(page, `var basePath = "/wled";`) {
		t.Error("page lacks the base path for its scripts")
	}
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`(?:href|src|action)="(/[^"]*)"`),
		regexp.MustCompile(`fetch\('(/[^']*)'`),
		regexp.MustCompile(`EventSource\('(/[^']*)'`),
	} {
		for _, m := range re.FindAllStringSubmatch(page, -1) {
			if !strings.HasPrefix(m[1], "/wled/") {
				t.Errorf("local URL without the base path: %s", m[0])
			}
		}
	}
	for _, want := range []string{`href="/wled/api/state"`, `href="/wled/api/reset"`, `fetch(basePath + '/json/state/events')`} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
}

// homerun2-portal shows version and commit from /health.
func TestHealthCarriesVersion(t *testing.T) {
	s := NewServer("v1.2.3", "abc1234def", "2026-01-01")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", http.NoBody))
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" || body["version"] != "v1.2.3" || body["commit"] != "abc1234def" {
		t.Errorf("unexpected health body: %v", body)
	}
}
