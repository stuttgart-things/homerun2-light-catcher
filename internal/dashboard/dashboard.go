package dashboard

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Replayer plays a timeline event's effect again. It returns false when the
// current profile matches no effect for it.
type Replayer func(ev LightEvent) bool

// replayLimit is how many replays a minute the dashboard accepts, so a stuck
// click or a script cannot keep the strip flashing.
const replayLimit = 30

// Handler serves the HTMX dashboard and API endpoints.
type Handler struct {
	tracker *EventTracker
	version string
	commit  string
	date    string

	// basePath prefixes every URL the page hands out (BASE_PATH, e.g.
	// "/light" behind homerun2-portal). Empty serves at the root.
	basePath string

	replay  Replayer
	mu      sync.Mutex
	replays []time.Time
	now     func() time.Time
}

// NewHandler creates a dashboard handler.
func NewHandler(tracker *EventTracker, version, commit, date string) *Handler {
	return &Handler{
		tracker: tracker,
		version: version,
		commit:  commit,
		date:    date,
		now:     time.Now,
	}
}

// SetReplayer enables the ▶ button of the timeline and POST
// /api/events/{id}/replay (#84).
func (h *Handler) SetReplayer(r Replayer) {
	h.replay = r
}

// SetBasePath sets the URL prefix the dashboard is served under, as
// config.ParseBasePath normalised it ("/light", or "" for the root). The
// routes stay at the root of the mux; handlers.Mount strips the prefix.
func (h *Handler) SetBasePath(basePath string) {
	h.basePath = basePath
}

// RegisterRoutes registers dashboard routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", h.handleDashboard)
	mux.HandleFunc("/api/events", h.handleEvents)
	mux.HandleFunc("/timeline", h.handleTimeline)
	mux.HandleFunc("POST /api/events/{id}/replay", h.handleReplay)
	mux.HandleFunc("/ui/manifest.json", h.handleManifest)
	mux.HandleFunc("/ui/fragment/events", h.handleEventsFragment)
}

// handleTimeline serves the rendered timeline, which the page polls.
func (h *Handler) handleTimeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("X-Event-Count", strconv.Itoa(h.tracker.Count()))
	fmt.Fprint(w, h.renderTimeline())
}

// handleReplay plays an event's effect again (#84).
func (h *Handler) handleReplay(w http.ResponseWriter, r *http.Request) {
	if h.replay == nil {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad event id", http.StatusBadRequest)
		return
	}
	ev, ok := h.tracker.Get(id)
	if !ok || !ev.On {
		http.Error(w, fmt.Sprintf("no event %d to play again", id), http.StatusNotFound)
		return
	}
	if wait := h.acquire(); wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		http.Error(w, fmt.Sprintf("rate limit: %d replays per minute", replayLimit), http.StatusTooManyRequests)
		return
	}
	if !h.replay(ev) {
		http.Error(w, "the profile matches no effect for this event any more", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// acquire takes a replay slot, or returns how long until one frees up.
func (h *Handler) acquire() time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	for len(h.replays) > 0 && now.Sub(h.replays[0]) >= time.Minute {
		h.replays = h.replays[1:]
	}
	if len(h.replays) >= replayLimit {
		return time.Minute - now.Sub(h.replays[0])
	}
	h.replays = append(h.replays, now)
	return 0
}

func (h *Handler) handleEvents(w http.ResponseWriter, r *http.Request) {
	events := h.tracker.Events()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"events": events,
		"count":  h.tracker.Count(),
	})
}

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, h.generateHTML())
}

func severityDotClass(severity string) string {
	switch strings.ToUpper(severity) {
	case "ERROR":
		return "error"
	case "WARNING":
		return "warning"
	case "SUCCESS":
		return "success"
	default:
		return "info"
	}
}

func (h *Handler) generateHTML() string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html lang="en" data-theme="dark"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>HOMERUN² Light Catcher</title>
<link href="https://fonts.googleapis.com/css2?family=Press+Start+2P&display=swap" rel="stylesheet">
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: 'Segoe UI', 'Roboto', Arial, sans-serif;
    background-color: #1e293b;
    color: #e0e0e0;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }
  .header-bar { background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 50%, #a855f7 100%); color: #f8fafc; padding: 1.8rem 1.5rem; display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 3px solid #f97316; }
  .header-bar h1 { margin: 0; font-family: 'Press Start 2P', cursive; font-size: 2.2rem; color: #ffffff; letter-spacing: 0.08em; text-shadow: 3px 3px 0px rgba(0,0,0,0.3); }
  .header-bar .subtitle { font-family: 'Press Start 2P', cursive; font-size: 0.7rem; color: #fbbf24; margin-top: 0.5rem; letter-spacing: 0.12em; text-transform: uppercase; }
  .header-bar .actions { display: flex; gap: 0.75rem; align-items: center; }
  .header-bar .actions a { color: #e2e8f0; font-size: 0.85rem; text-decoration: none; }
  .header-bar .actions a:hover { color: #f8fafc; }
  .main-content { flex: 1; padding: 20px; }
  .stats { display: flex; gap: 16px; justify-content: center; margin-bottom: 24px; }
  .stat-card { background: #0f172a; border: 1px solid #334155; border-radius: 10px; padding: 16px 24px; text-align: center; min-width: 120px; }
  .stat-card .number { font-size: 28px; font-weight: bold; color: #818cf8; }
  .stat-card .label { font-size: 12px; color: #64748b; text-transform: uppercase; margin-top: 4px; }
  .timeline-section { max-width: 800px; margin: 0 auto; }
  .timeline-title { font-size: 16px; font-weight: bold; color: #818cf8; margin-bottom: 10px; text-transform: uppercase; letter-spacing: 1px; }
  #timeline { background-color: #0f172a; border-radius: 8px; border: 1px solid #334155; padding: 12px; max-height: 500px; overflow-y: auto; font-size: 13px; }
  #timeline::-webkit-scrollbar { width: 6px; }
  #timeline::-webkit-scrollbar-track { background: #0f172a; border-radius: 3px; }
  #timeline::-webkit-scrollbar-thumb { background: #334155; border-radius: 3px; }
  .event-row { display: flex; align-items: center; gap: 10px; padding: 6px 8px; border-bottom: 1px solid #1e293b; }
  .event-row:last-child { border-bottom: none; }
  .event-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
  .event-dot.error { background-color: #ff4444; box-shadow: 0 0 6px #ff4444; }
  .event-dot.warning { background-color: #f97316; box-shadow: 0 0 6px #f97316; }
  .event-dot.success { background-color: #4ade80; box-shadow: 0 0 6px #4ade80; }
  .event-dot.info { background-color: #60a5fa; box-shadow: 0 0 6px #60a5fa; }
  .event-dot.off { background-color: #64748b; }
  .event-time { color: #64748b; font-family: 'Courier New', monospace; min-width: 65px; }
  .event-severity { font-weight: bold; min-width: 70px; text-transform: uppercase; font-size: 12px; }
  .event-severity.error { color: #ff4444; }
  .event-severity.warning { color: #f97316; }
  .event-severity.success { color: #4ade80; }
  .event-severity.info { color: #60a5fa; }
  .event-system { color: #818cf8; min-width: 100px; }
  .event-effect { color: #e2e8f0; }
  .event-tags { display: flex; flex-wrap: wrap; gap: 4px; }
  .event-tag { font-family: 'Courier New', monospace; font-size: 11px; color: #fbbf24; background: rgba(251,191,36,0.12); border: 1px solid rgba(251,191,36,0.35); border-radius: 4px; padding: 1px 6px; }
  .event-off { color: #64748b; font-style: italic; }
  details.event { border-bottom: 1px solid #1e293b; }
  details.event > summary { list-style: none; cursor: pointer; border-bottom: none; }
  details.event > summary::-webkit-details-marker { display: none; }
  details.event > summary:hover { background: #1e293b; }
  details.event[open] > summary { background: #1e293b; }
  .event-title { color: #f8fafc; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .event-replay { color: #a855f7; margin-right: 4px; }
  .event-detail { padding: 8px 12px 12px 30px; color: #cbd5e1; font-size: 12px; }
  .detail-message { white-space: pre-wrap; margin-bottom: 8px; color: #e2e8f0; }
  .event-detail dl { display: grid; grid-template-columns: 70px 1fr; gap: 2px 10px; }
  .event-detail dt { color: #64748b; }
  .event-detail dd { font-family: 'Courier New', monospace; word-break: break-all; }
  .event-detail a { color: #818cf8; }
  button.replay { background: none; border: 1px solid #334155; color: #a855f7; border-radius: 4px; padding: 1px 8px; cursor: pointer; font-size: 12px; }
  button.replay:hover { border-color: #a855f7; }
  button.replay.ok { color: #4ade80; border-color: #4ade80; }
  button.replay.fail { color: #ff4444; border-color: #ff4444; }
  .preview { max-width: 800px; margin: 0 auto 24px; background: #0f172a; border: 1px solid #334155; border-radius: 10px; padding: 14px 16px; }
  .preview-head { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; font-size: 13px; flex-wrap: wrap; }
  .preview-mode { font-weight: bold; text-transform: uppercase; letter-spacing: 1px; font-size: 12px; padding: 2px 8px; border-radius: 4px; }
  .preview-mode.live { color: #4ade80; background: rgba(74,222,128,0.12); }
  .preview-mode.replay { color: #a855f7; background: rgba(168,85,247,0.15); }
  .preview-label { color: #e2e8f0; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .preview-live { background: none; border: 1px solid #334155; color: #4ade80; border-radius: 4px; padding: 1px 8px; cursor: pointer; font-size: 12px; }
  .preview-live[hidden] { display: none; }
  .strip { display: flex; gap: 4px; padding: 10px; background: #020617; border-radius: 8px; }
  .led { flex: 1; aspect-ratio: 1; border-radius: 50%; background: #111827; }
  .preview-info { margin-top: 8px; color: #64748b; font-size: 12px; }
  .swatches { display: inline-flex; gap: 3px; vertical-align: middle; margin-left: 6px; }
  .swatch { width: 12px; height: 12px; border-radius: 50%; border: 1px solid rgba(255,255,255,0.25); }
  details.event.selected > summary { background: rgba(168,85,247,0.15); box-shadow: inset 3px 0 0 #a855f7; }
  .empty-state { text-align: center; color: #64748b; padding: 40px; font-size: 14px; }
  .build-footer { background: #0f172a; color: #475569; padding: 0.6rem 1.5rem; display: flex; gap: 1.5rem; font-size: 0.75rem; border-top: 1px solid #334155; }
  .build-footer .label { color: #64748b; }
  .build-footer .value { color: #818cf8; }
</style>
</head><body>
<div class="header-bar">
  <div>
    <h1>HOMERUN²</h1>
    <div class="subtitle">light catcher dashboard</div>
  </div>
  <div class="actions">
`)
	base := html.EscapeString(h.basePath)
	fmt.Fprintf(&sb, `    <a href="%s/api/events">API Events</a>
    <a href="%s/health">Health</a>
`, base, base)
	sb.WriteString(`  </div>
</div>
<div class="main-content">
  <div class="stats">
    <div class="stat-card"><div class="number" id="event-count">`)

	fmt.Fprintf(&sb, "%d", h.tracker.Count())

	sb.WriteString(`</div><div class="label">Events</div></div>
  </div>
  <div class="preview" id="preview">
    <div class="preview-head">
      <span class="preview-mode live" id="preview-mode">Live</span>
      <span class="preview-label" id="preview-label">Light off</span>
      <button class="preview-live" id="preview-live" hidden title="Back to the live state">● Live</button>
    </div>
    <div class="strip" id="strip"></div>
    <div class="preview-info" id="preview-info">Click an event in the timeline to see its effect again.</div>
  </div>
  <div class="timeline-section">
    <div class="timeline-title">Light Event Timeline</div>
    <div id="timeline">`)

	sb.WriteString(h.renderTimeline())

	basePathJS, _ := json.Marshal(h.basePath)
	fmt.Fprintf(&sb, `</div></div></div>
<script>
// URL prefix behind homerun2-portal (BASE_PATH), "" at the root.
var basePath = %s;
`, basePathJS)
	sb.WriteString(`// The server renders the timeline (one place that escapes); this polls it and
// keeps open details open across refreshes (#84).
var openIds = {};
function updateDashboard() {
  fetch(basePath + '/timeline')
    .then(function(r) {
      var count = r.headers.get('X-Event-Count');
      if (count !== null) document.getElementById('event-count').textContent = count;
      return r.text();
    })
    .then(function(html) {
      var tl = document.getElementById('timeline');
      tl.querySelectorAll('details[open]').forEach(function(d) { openIds[d.dataset.id] = true; });
      tl.innerHTML = html;
      tl.querySelectorAll('details').forEach(function(d) { if (openIds[d.dataset.id]) d.open = true; });
      preview.refresh();
    })
    .catch(function(err) { console.error('Error:', err); });
}
document.getElementById('timeline').addEventListener('toggle', function(e) {
  if (e.target.tagName === 'DETAILS' && !e.target.open) delete openIds[e.target.dataset.id];
}, true);
document.getElementById('timeline').addEventListener('click', function(e) {
  var btn = e.target.closest('button.replay');
  if (!btn) return;
  e.preventDefault();
  e.stopPropagation();
  btn.disabled = true;
  fetch(basePath + '/api/events/' + btn.dataset.id + '/replay', { method: 'POST' })
    .then(function(r) { btn.classList.add(r.ok ? 'ok' : 'fail'); return updateDashboard(); })
    .finally(function() { btn.disabled = false; });
});
setInterval(updateDashboard, 2000);
</script>
<script>`)
	sb.WriteString(previewJS)
	sb.WriteString(`</script>`)

	fmt.Fprintf(&sb, `<div class="build-footer">
  <div style="display:flex;gap:1.5rem">
    <div><span class="label">version</span> <span class="value">%s</span></div>
    <div><span class="label">commit</span> <span class="value">%s</span></div>
    <div><span class="label">built</span> <span class="value">%s</span></div>
  </div>
  <div style="margin-left:auto;display:flex;align-items:center;gap:0.5rem"><span class="label">a</span> <a href="https://github.com/stuttgart-things" target="_blank" style="color:#818cf8;text-decoration:none">stuttgart-things</a> <span class="label">project</span> <img src="https://raw.githubusercontent.com/stuttgart-things/docs/main/hugo/sthings-logo.png" alt="sthings" style="height:24px;"></div>
</div>
</body></html>`, h.version, shortCommit(h.commit), h.date)

	return sb.String()
}

// renderTimeline renders the timeline, newest first. Every field that comes
// from a message is escaped: titles, systems and tags are whatever a pitcher
// sent, and the page is served to every viewer (#84).
func (h *Handler) renderTimeline() string {
	events := h.tracker.Events()
	if len(events) == 0 {
		return `<div class="empty-state">No light events yet. Waiting for messages...</div>`
	}
	esc := html.EscapeString

	var sb strings.Builder
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if !ev.On {
			sb.WriteString(`<div class="event-row" data-off="1">`)
			sb.WriteString(`<span class="event-dot off"></span>`)
			fmt.Fprintf(&sb, `<span class="event-time">%s</span>`, esc(ev.Timestamp))
			sb.WriteString(`<span class="event-off">Light turned off</span>`)
			sb.WriteString(`</div>`)
			continue
		}

		cls := severityDotClass(ev.Severity)
		fmt.Fprintf(&sb, `<details class="event" data-id="%d"%s><summary class="event-row">`, ev.ID, lookAttrs(ev))
		fmt.Fprintf(&sb, `<span class="event-dot %s"></span>`, cls)
		fmt.Fprintf(&sb, `<span class="event-time">%s</span>`, esc(ev.Timestamp))
		fmt.Fprintf(&sb, `<span class="event-severity %s">%s</span>`, cls, esc(ev.Severity))
		fmt.Fprintf(&sb, `<span class="event-system">%s</span>`, esc(ev.System))
		replayMark := ""
		if ev.ReplayOf > 0 {
			replayMark = `<span class="event-replay" title="played again">↻</span>`
		}
		fmt.Fprintf(&sb, `<span class="event-title">%s%s</span>`, replayMark, esc(ev.Title))
		look := ev.Color
		if look == "" {
			look = ev.Palette
		}
		fmt.Fprintf(&sb, `<span class="event-effect">%s / %s%s</span>`, esc(ev.Effect), esc(look), swatches(ev.RGB))
		if len(ev.Tags) > 0 {
			sb.WriteString(`<span class="event-tags" title="matched tags">`)
			for _, tag := range ev.Tags {
				fmt.Fprintf(&sb, `<span class="event-tag">%s</span>`, esc(tag))
			}
			sb.WriteString(`</span>`)
		}
		if h.replay != nil {
			fmt.Fprintf(&sb, `<button class="replay" data-id="%d" title="Play again on the strip">▶</button>`, ev.ID)
		}
		sb.WriteString(`</summary><div class="event-detail">`)
		if ev.Message != "" {
			fmt.Fprintf(&sb, `<div class="detail-message">%s</div>`, esc(ev.Message))
		}
		sb.WriteString(`<dl>`)
		if ev.Author != "" {
			fmt.Fprintf(&sb, `<dt>author</dt><dd>%s</dd>`, esc(ev.Author))
		}
		if ev.MessageTags != "" {
			fmt.Fprintf(&sb, `<dt>tags</dt><dd>%s</dd>`, esc(ev.MessageTags))
		}
		if u := safeURL(ev.URL); u != "" {
			fmt.Fprintf(&sb, `<dt>link</dt><dd><a href="%s" target="_blank" rel="noopener">%s</a></dd>`, esc(u), esc(u))
		}
		fmt.Fprintf(&sb, `<dt>endpoint</dt><dd>%s</dd>`, esc(ev.Endpoint))
		sb.WriteString(`</dl></div></details>`)
	}
	return sb.String()
}

// lookAttrs renders what the preview strip needs to play an event's effect
// as data attributes. Every value is escaped: the names come from the profile.
func lookAttrs(ev LightEvent) string {
	rgb, _ := json.Marshal(ev.RGB)
	if ev.RGB == nil {
		rgb = []byte("[]")
	}
	var sb strings.Builder
	for _, a := range [][2]string{
		{"fx", ev.Effect}, {"rgb", string(rgb)}, {"pal", ev.Palette},
		{"sx", strconv.Itoa(ev.Speed)}, {"ix", strconv.Itoa(ev.Intensity)}, {"label", previewLabel(ev)},
	} {
		sb.WriteString(` data-` + a[0] + `="` + html.EscapeString(a[1]) + `"`)
	}
	return sb.String()
}

// previewLabel names an event above the preview strip.
func previewLabel(ev LightEvent) string {
	parts := []string{ev.Timestamp}
	for _, p := range []string{ev.Severity, ev.System, ev.Title} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, " · ")
}

// swatches renders a dot per color the effect was sent with, at most eight.
func swatches(rgb [][3]int) string {
	if len(rgb) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(`<span class="swatches">`)
	for i, c := range rgb {
		if i == 8 {
			break
		}
		fmt.Fprintf(&sb, `<span class="swatch" style="background:rgb(%d,%d,%d)"></span>`, c[0], c[1], c[2])
	}
	sb.WriteString(`</span>`)
	return sb.String()
}

// safeURL returns u if it is an http(s) link, else "": a message's URL
// becomes an href, and javascript: must not.
func safeURL(u string) string {
	lower := strings.ToLower(strings.TrimSpace(u))
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
		return strings.TrimSpace(u)
	}
	return ""
}
