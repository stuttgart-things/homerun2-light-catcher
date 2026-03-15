package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Handler serves the HTMX dashboard and API endpoints.
type Handler struct {
	tracker *EventTracker
	version string
	commit  string
	date    string
}

// NewHandler creates a dashboard handler.
func NewHandler(tracker *EventTracker, version, commit, date string) *Handler {
	return &Handler{
		tracker: tracker,
		version: version,
		commit:  commit,
		date:    date,
	}
}

// RegisterRoutes registers dashboard routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", h.handleDashboard)
	mux.HandleFunc("/api/events", h.handleEvents)
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
	events := h.tracker.Events()

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
  .event-off { color: #64748b; font-style: italic; }
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
    <a href="/api/events">API Events</a>
    <a href="/health">Health</a>
  </div>
</div>
<div class="main-content">
  <div class="stats">
    <div class="stat-card"><div class="number" id="event-count">`)

	fmt.Fprintf(&sb, "%d", h.tracker.Count())

	sb.WriteString(`</div><div class="label">Events</div></div>
  </div>
  <div class="timeline-section">
    <div class="timeline-title">Light Event Timeline</div>
    <div id="timeline">`)

	if len(events) == 0 {
		sb.WriteString(`<div class="empty-state">No light events yet. Waiting for messages...</div>`)
	} else {
		for i := len(events) - 1; i >= 0; i-- {
			ev := events[i]
			sb.WriteString(`<div class="event-row">`)
			if ev.On {
				cls := severityDotClass(ev.Severity)
				fmt.Fprintf(&sb, `<span class="event-dot %s"></span>`, cls)
				fmt.Fprintf(&sb, `<span class="event-time">%s</span>`, ev.Timestamp)
				fmt.Fprintf(&sb, `<span class="event-severity %s">%s</span>`, cls, ev.Severity)
				fmt.Fprintf(&sb, `<span class="event-system">%s</span>`, ev.System)
				fmt.Fprintf(&sb, `<span class="event-effect">%s / %s</span>`, ev.Effect, ev.Color)
			} else {
				sb.WriteString(`<span class="event-dot off"></span>`)
				fmt.Fprintf(&sb, `<span class="event-time">%s</span>`, ev.Timestamp)
				sb.WriteString(`<span class="event-off">Light turned off</span>`)
			}
			sb.WriteString(`</div>`)
		}
	}

	sb.WriteString(`</div></div></div>
<script>
function updateDashboard() {
  fetch('/api/events')
    .then(function(r) { return r.json(); })
    .then(function(data) {
      document.getElementById('event-count').textContent = data.count;
      var events = data.events;
      var tl = document.getElementById('timeline');
      if (events.length === 0) {
        tl.innerHTML = '<div class="empty-state">No light events yet. Waiting for messages...</div>';
        return;
      }
      var html = '';
      for (var i = events.length - 1; i >= 0; i--) {
        var ev = events[i];
        html += '<div class="event-row">';
        if (ev.on) {
          var sev = (ev.severity || 'info').toUpperCase();
          var cls = sev === 'ERROR' ? 'error' : sev === 'WARNING' ? 'warning' : sev === 'SUCCESS' ? 'success' : 'info';
          html += '<span class="event-dot ' + cls + '"></span>';
          html += '<span class="event-time">' + ev.timestamp + '</span>';
          html += '<span class="event-severity ' + cls + '">' + sev + '</span>';
          html += '<span class="event-system">' + (ev.system || '') + '</span>';
          html += '<span class="event-effect">' + (ev.effect || '') + ' / ' + (ev.color || '') + '</span>';
        } else {
          html += '<span class="event-dot off"></span>';
          html += '<span class="event-time">' + ev.timestamp + '</span>';
          html += '<span class="event-off">Light turned off</span>';
        }
        html += '</div>';
      }
      tl.innerHTML = html;
    })
    .catch(function(err) { console.error('Error:', err); });
}
setInterval(updateDashboard, 2000);
</script>`)

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
