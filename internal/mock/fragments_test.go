package mock

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManifestListsTheStateFragment(t *testing.T) {
	s := NewServer("v1.2.3", "abc1234", "2026-01-01")
	s.SetBasePath("/wled")

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wled/ui/manifest.json", http.NoBody))
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("manifest: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	var m Manifest
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	if m.Title != "WLED Mock" || m.Version != "v1.2.3" || m.BasePath != "/wled" || len(m.Fragments) != 1 {
		t.Fatalf("manifest = %+v", m)
	}
	f := m.Fragments[0]
	if f.ID != "state" || f.URL != "/wled/ui/fragment/state" || f.RefreshSeconds < 2 {
		t.Errorf("fragment = %+v", f)
	}

	// The fragment answers at the URL the manifest names.
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, f.URL, http.NoBody))
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Errorf("fragment: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
}

func TestStateFragmentOff(t *testing.T) {
	s := NewServer("test", "", "")
	s.SetBasePath("/wled")

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wled/ui/fragment/state", http.NoBody))
	body := rec.Body.String()
	assertBareFragment(t, body)
	for _, want := range []string{"Light is off", `href="/wled/"`, "Open simulator"} {
		if !strings.Contains(body, want) {
			t.Errorf("fragment lacks %s:\n%s", want, body)
		}
	}
	if strings.Contains(body, "hr-time") {
		t.Error("a state never changed has no time")
	}
}

func TestStateFragmentOn(t *testing.T) {
	s := NewServer("test", "", "")
	s.SetBasePath("/wled")
	// The effect list can come from a file, so names are escaped.
	s.effectNames[1] = `<b>Blink</b>`
	h := s.Handler()

	post := `{"on":true,"bri":255,"seg":[` +
		`{"fx":1,"pal":11,"col":[[255,0,0]]},` +
		`{"fx":0,"pal":0,"col":[[0,0,255]]}]}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/json/state", strings.NewReader(post)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /json/state: %d %s", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wled/ui/fragment/state", http.NoBody))
	body := rec.Body.String()
	assertBareFragment(t, body)
	for _, want := range []string{
		`<span class="hr-time">`,
		`<span class="hr-title">On</span><span class="hr-muted">brightness 255/255 (100%)</span>`,
		`<span class="hr-dot" style="background: rgb(255, 0, 0)"></span><span class="hr-title">&lt;b&gt;Blink&lt;/b&gt;</span><span class="hr-system">Rainbow</span>`,
		`<span class="hr-dot" style="background: rgb(0, 0, 255)"></span><span class="hr-title">Solid</span></li>`, // palette 0 is not shown
		`href="/wled/"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("fragment lacks %s:\n%s", want, body)
		}
	}
	if strings.Contains(body, "Light is off") {
		t.Error("the light is on")
	}
}

func TestStateFragmentCapsTheSegments(t *testing.T) {
	s := NewServer("test", "", "")
	s.state.On = true
	s.state.Seg = make([]Segment, 12)
	body := s.renderStateFragment()
	if n := strings.Count(body, `<span class="hr-dot"`); n != stateFragmentSegments {
		t.Errorf("%d segments listed, want %d", n, stateFragmentSegments)
	}
	if !strings.Contains(body, "+4 more segments") || !strings.Contains(body, `href="/"`) {
		t.Errorf("fragment = %s", body)
	}
}

func TestFragmentsAtTheRootWithoutBasePath(t *testing.T) {
	h := NewServer("test", "", "").Handler()
	for _, path := range []string{"/ui/manifest.json", "/ui/fragment/state"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: %d", path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/manifest.json", http.NoBody))
	var m Manifest
	_ = json.NewDecoder(rec.Body).Decode(&m)
	if len(m.Fragments) != 1 || m.Fragments[0].URL != "/ui/fragment/state" {
		t.Errorf("manifest = %+v", m)
	}
}

// assertBareFragment checks the contract: no document, no script, only the
// portal's hr-* classes.
func assertBareFragment(t *testing.T, body string) {
	t.Helper()
	for _, banned := range []string{"<html", "<head", "<script", "<style"} {
		if strings.Contains(body, banned) {
			t.Errorf("a fragment is bare HTML, found %s", banned)
		}
	}
	for _, attr := range strings.Split(body, `class="`)[1:] {
		classes, _, _ := strings.Cut(attr, `"`)
		for _, c := range strings.Fields(classes) {
			if !strings.HasPrefix(c, "hr-") {
				t.Errorf("class %q is not one of the portal's", c)
			}
		}
	}
}
