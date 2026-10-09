package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Tests for #84: the timeline shows the triggering message, escapes every
// field, and replays an effect.

func record(t *EventTracker, ev LightEvent) int {
	ev.On = true
	return t.RecordEvent(ev)
}

func TestTimelineShowsAndEscapesTheMessage(t *testing.T) {
	tracker := NewEventTracker()
	record(tracker, LightEvent{
		Severity: "warning", System: `<script>alert(1)</script>`, Effect: "Breathe", Color: "orange",
		Title: "PR <b>#1</b>", Message: "opened & ready", Author: "dev", URL: "https://github.com/x/pull/1",
		MessageTags: "github,pull_request",
	})
	h := NewHandler(tracker, "v", "c", "d")

	html := h.renderTimeline()
	for _, want := range []string{
		`&lt;script&gt;alert(1)&lt;/script&gt;`,
		`PR &lt;b&gt;#1&lt;/b&gt;`,
		`opened &amp; ready`,
		`<a href="https://github.com/x/pull/1"`,
		`<dt>tags</dt><dd>github,pull_request</dd>`,
		`<details class="event" data-id="1">`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("timeline lacks %q", want)
		}
	}
	if strings.Contains(html, "<script>") {
		t.Error("unescaped script tag in the timeline")
	}
	if strings.Contains(html, `button class="replay"`) {
		t.Error("no replay button without a replayer")
	}
}

func TestTimelineDropsNonHTTPLinks(t *testing.T) {
	tracker := NewEventTracker()
	record(tracker, LightEvent{Severity: "info", URL: "javascript:alert(1)"})

	if strings.Contains(NewHandler(tracker, "v", "c", "d").renderTimeline(), "javascript:") {
		t.Error("a javascript: URL must not become a link")
	}
}

func TestReplayEndpoint(t *testing.T) {
	tracker := NewEventTracker()
	id := record(tracker, LightEvent{Severity: "error", System: "ci", Title: "red"})
	off := tracker.RecordEvent(LightEvent{On: false})

	var replayed []LightEvent
	matches := true
	h := NewHandler(tracker, "v", "c", "d")
	h.SetReplayer(func(ev LightEvent) bool { replayed = append(replayed, ev); return matches })
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	post := func(path string) int {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		return rec.Code
	}

	if code := post("/api/events/1/replay"); code != http.StatusAccepted {
		t.Fatalf("replay: got %d", code)
	}
	if len(replayed) != 1 || replayed[0].ID != id || replayed[0].Title != "red" {
		t.Errorf("replayer got %+v", replayed)
	}
	if code := post("/api/events/99/replay"); code != http.StatusNotFound {
		t.Errorf("unknown id: got %d", code)
	}
	if code := post("/api/events/" + strconv.Itoa(off) + "/replay"); code != http.StatusNotFound {
		t.Errorf("an off event is not replayable: got %d", code)
	}
	if code := post("/api/events/x/replay"); code != http.StatusBadRequest {
		t.Errorf("bad id: got %d", code)
	}
	matches = false
	if code := post("/api/events/1/replay"); code != http.StatusConflict {
		t.Errorf("no matching effect: got %d", code)
	}
	if !strings.Contains(h.renderTimeline(), `<button class="replay" data-id="1"`) {
		t.Error("expected a replay button with a replayer")
	}
}

func TestReplayRateLimit(t *testing.T) {
	tracker := NewEventTracker()
	record(tracker, LightEvent{Severity: "error"})
	h := NewHandler(tracker, "v", "c", "d")
	clock := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return clock }
	h.SetReplayer(func(LightEvent) bool { return true })
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	codes := map[int]int{}
	for i := 0; i < replayLimit+1; i++ {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/events/1/replay", nil))
		codes[rec.Code]++
	}
	if codes[http.StatusAccepted] != replayLimit || codes[http.StatusTooManyRequests] != 1 {
		t.Errorf("got %v", codes)
	}

	clock = clock.Add(time.Minute)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/events/1/replay", nil))
	if rec.Code != http.StatusAccepted {
		t.Errorf("after a minute: got %d", rec.Code)
	}
}

func TestTrackerGet(t *testing.T) {
	tracker := NewEventTracker()
	for i := 0; i < 105; i++ {
		record(tracker, LightEvent{Title: strconv.Itoa(i + 1)})
	}
	if _, ok := tracker.Get(5); ok {
		t.Error("event 5 has left the 100-slot buffer")
	}
	if ev, ok := tracker.Get(105); !ok || ev.Title != "105" {
		t.Errorf("event 105: %+v %v", ev, ok)
	}
	if ev, ok := tracker.Get(6); !ok || ev.Title != "6" {
		t.Errorf("event 6: %+v %v", ev, ok)
	}
}
