package dashboard

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
)

// Micro-frontend fragments for homerun2-portal. The portal reads
// <base>/ui/manifest.json to learn which fragments a service offers and loads
// each <base>/ui/fragment/<id> with htmx into its overview. A fragment is bare
// HTML styled only by the portal's hr-* classes (static/homerun2.css in
// homerun2-portal); every URL in it carries the base path, because it is
// served on the portal's origin.

const (
	eventsFragmentLimit   = 8
	eventsFragmentRefresh = 5
)

// Fragment is one entry of the manifest.
type Fragment struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	URL            string `json:"url"`
	RefreshSeconds int    `json:"refreshSeconds"`
}

// Manifest lists the fragments the service offers to the portal.
type Manifest struct {
	Name      string     `json:"name"`
	Title     string     `json:"title"`
	Version   string     `json:"version"`
	BasePath  string     `json:"basePath"`
	Fragments []Fragment `json:"fragments"`
}

// handleManifest serves GET /ui/manifest.json.
func (h *Handler) handleManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Manifest{
		Name:     "homerun2-light-catcher",
		Title:    "Light Catcher",
		Version:  h.version,
		BasePath: h.basePath,
		Fragments: []Fragment{{
			ID:             "events",
			Title:          "Latest light effects",
			URL:            h.basePath + "/ui/fragment/events",
			RefreshSeconds: eventsFragmentRefresh,
		}},
	})
}

// handleEventsFragment serves GET /ui/fragment/events.
func (h *Handler) handleEventsFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, h.renderEventsFragment())
}

// renderEventsFragment renders the newest light events as an hr-list. Every
// field is escaped: titles and systems are whatever a pitcher sent, and the
// portal shows the fragment to every viewer.
func (h *Handler) renderEventsFragment() string {
	esc := html.EscapeString
	events := h.tracker.Events()

	var sb strings.Builder
	sb.WriteString(`<div class="hr-frag"><ul class="hr-list">`)
	if len(events) == 0 {
		sb.WriteString(`<li class="hr-item hr-muted">No light events yet.</li>`)
	}
	for i := len(events) - 1; i >= 0 && i >= len(events)-eventsFragmentLimit; i-- {
		ev := events[i]
		sb.WriteString(`<li class="hr-item">`)
		fmt.Fprintf(&sb, `<span class="hr-time">%s</span>`, esc(ev.Timestamp))
		if !ev.On {
			sb.WriteString(`<span class="hr-dot"></span><span class="hr-title hr-muted">Light turned off</span></li>`)
			continue
		}
		fmt.Fprintf(&sb, `<span class="hr-dot"%s></span>`, dotStyle(ev.RGB))
		fmt.Fprintf(&sb, `<span class="hr-sev hr-sev-%s">%s</span>`, severityDotClass(ev.Severity), esc(strings.ToUpper(ev.Severity)))
		fmt.Fprintf(&sb, `<span class="hr-system">%s</span>`, esc(ev.System))
		title := ev.Title
		if title == "" {
			title = ev.Effect
		}
		fmt.Fprintf(&sb, `<span class="hr-title">%s</span>`, esc(title))
		if ev.Title != "" && ev.Effect != "" {
			fmt.Fprintf(&sb, `<span class="hr-system">%s</span>`, esc(ev.Effect))
		}
		sb.WriteString(`</li>`)
	}
	fmt.Fprintf(&sb, `</ul><a class="hr-frag-more" href="%s/">Open dashboard →</a></div>`, esc(h.basePath))
	return sb.String()
}

// dotStyle colors the dot with the first color the strip was sent.
func dotStyle(rgb [][3]int) string {
	if len(rgb) == 0 {
		return ""
	}
	c := rgb[0]
	return fmt.Sprintf(` style="background:rgb(%d,%d,%d)"`, c[0], c[1], c[2])
}
