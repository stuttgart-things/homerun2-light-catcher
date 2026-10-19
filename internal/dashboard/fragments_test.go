package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManifestListsTheEventsFragment(t *testing.T) {
	h := NewHandler(NewEventTracker(), "v1.2.3", "abc1234", "2026-01-01")
	h.SetBasePath("/light")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/manifest.json", http.NoBody))
	var m Manifest
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	if m.Version != "v1.2.3" || m.BasePath != "/light" || len(m.Fragments) != 1 {
		t.Fatalf("manifest = %+v", m)
	}
	if f := m.Fragments[0]; f.ID != "events" || f.URL != "/light/ui/fragment/events" || f.RefreshSeconds < 2 {
		t.Errorf("fragment = %+v", f)
	}
}

func TestEventsFragment(t *testing.T) {
	tracker := NewEventTracker()
	for range 10 {
		tracker.RecordEvent(LightEvent{Severity: "success", System: "old", Effect: "Solid", On: true})
	}
	tracker.RecordOff("http://wled")
	tracker.RecordEvent(LightEvent{
		Severity: "warning", System: "nas", Effect: "Blink", On: true,
		Title: `<b>disk</b> full`, RGB: [][3]int{{255, 0, 0}},
	})
	h := NewHandler(tracker, "test", "abc1234", "2026-01-01")
	h.SetBasePath("/light")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/fragment/events", http.NoBody))
	body := rec.Body.String()

	if strings.Contains(body, "<html") || strings.Contains(body, "<script") {
		t.Fatal("a fragment is bare HTML")
	}
	if n := strings.Count(body, `<li class="hr-item">`); n != eventsFragmentLimit {
		t.Errorf("%d items, want %d", n, eventsFragmentLimit)
	}
	for _, want := range []string{
		`<span class="hr-dot" style="background:rgb(255,0,0)"></span><span class="hr-sev hr-sev-warning">WARNING</span>`,
		"&lt;b&gt;disk&lt;/b&gt; full", // titles come from messages and are escaped
		`<span class="hr-system">Blink</span>`,
		"Light turned off",
		`href="/light/"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("fragment lacks %s:\n%s", want, body)
		}
	}
	if i := strings.Index(body, "disk"); i < 0 || i > strings.Index(body, "turned off") {
		t.Error("the newest event is not first")
	}
}

func TestEventsFragmentEmpty(t *testing.T) {
	body := NewHandler(NewEventTracker(), "test", "", "").renderEventsFragment()
	if !strings.Contains(body, "No light events yet.") || !strings.Contains(body, `href="/"`) {
		t.Errorf("empty fragment = %s", body)
	}
}
