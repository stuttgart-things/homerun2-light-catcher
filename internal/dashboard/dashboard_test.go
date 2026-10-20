package dashboard

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerateHTML_ShowsMatchedTags(t *testing.T) {
	tracker := NewEventTracker()
	tracker.Record("info", "tabletennis", "Solid", "blue", "http://wled", []string{"transition=point", "side=<a>"})
	tracker.Record("info", "github", "DJ Light", "ocean", "http://wled", nil)

	page := NewHandler(tracker, "test", "abc1234", "2026-01-01").generateHTML()
	// Only inspect the server-rendered timeline, not the JS that re-renders it.
	_, body, _ := strings.Cut(page, "<body>")
	html, _, _ := strings.Cut(body, "<script>")

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

// Behind homerun2-portal the page is served under BASE_PATH, and every URL it
// hands out must carry it: the portal forwards only /light/... to this service.
func TestGenerateHTML_CarriesBasePath(t *testing.T) {
	tracker := NewEventTracker()
	tracker.Record("info", "github", "DJ Light", "ocean", "http://wled", nil)
	h := NewHandler(tracker, "test", "abc1234", "2026-01-01")
	h.SetReplayer(func(LightEvent) bool { return true })
	h.SetBasePath("/light")
	page := h.generateHTML()

	if !strings.Contains(page, `var basePath = "/light";`) {
		t.Error("page lacks the base path for its scripts")
	}
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`(?:href|src|action)="(/[^"]*)"`),
		regexp.MustCompile(`fetch\('(/[^']*)'`),
	} {
		for _, m := range re.FindAllStringSubmatch(page, -1) {
			if !strings.HasPrefix(m[1], "/light/") {
				t.Errorf("local URL without the base path: %s", m[0])
			}
		}
	}
	for _, want := range []string{`href="/light/api/events"`, `href="/light/health"`, `fetch(basePath + '/timeline')`, `fetch(basePath + '/api/events/'`} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
}

func TestGenerateHTML_RootWithoutBasePath(t *testing.T) {
	page := NewHandler(NewEventTracker(), "test", "abc1234", "2026-01-01").generateHTML()
	for _, want := range []string{`var basePath = "";`, `href="/api/events"`, `href="/health"`} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
}
