package dashboard

import (
	"strings"
	"testing"
)

func TestGenerateHTML_ShowsMatchedTags(t *testing.T) {
	tracker := NewEventTracker()
	tracker.Record("info", "tabletennis", "Solid", "blue", "http://wled", []string{"transition=point", "side=<a>"})
	tracker.Record("info", "github", "DJ Light", "ocean", "http://wled", nil)

	page := NewHandler(tracker, "test", "abc1234", "2026-01-01").generateHTML()
	// Only inspect the server-rendered timeline, not the JS that re-renders it.
	html, _, _ := strings.Cut(page, "<script>")

	if !strings.Contains(html, `<span class="event-tag">transition=point</span>`) {
		t.Error("expected matched tag in timeline")
	}
	if !strings.Contains(html, `<span class="event-tag">side=&lt;a&gt;</span>`) {
		t.Error("expected matched tag to be HTML-escaped")
	}
	if got := strings.Count(html, `<span class="event-tags"`); got != 1 {
		t.Errorf("expected tags only on the event that matched tags, got %d tag groups", got)
	}
}

func TestRenderTimeline_CarriesTheLookForThePreview(t *testing.T) {
	tracker := NewEventTracker()
	tracker.RecordEvent(LightEvent{
		Severity: "info", System: "tabletennis", Effect: "Solid", Color: "blue", On: true,
		Title: `Pong "A" <b>`, RGB: [][3]int{{0, 0, 255}}, Speed: 128, Intensity: 255,
	})
	tracker.RecordEvent(LightEvent{Severity: "success", System: "ci", Effect: "Aurora", Palette: "Lava", On: true})
	tracker.RecordOff("http://wled")

	tl := NewHandler(tracker, "test", "abc1234", "2026-01-01").renderTimeline()

	for _, want := range []string{
		`data-fx="Solid" data-rgb="[[0,0,255]]" data-pal="" data-sx="128" data-ix="255"`,
		`Pong &#34;A&#34; &lt;b&gt;"`,
		`<span class="swatch" style="background:rgb(0,0,255)"></span>`,
		`data-fx="Aurora" data-rgb="[]" data-pal="Lava"`,
		`Aurora / Lava</span>`,
		`<div class="event-row" data-off="1">`,
	} {
		if !strings.Contains(tl, want) {
			t.Errorf("timeline lacks %s", want)
		}
	}
}
