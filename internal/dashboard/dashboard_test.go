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
