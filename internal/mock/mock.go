package mock

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/profile"
)

// WLEDState represents the state of the WLED light.
type WLEDState struct {
	On  bool      `json:"on"`
	Bri int       `json:"bri"`
	Seg []Segment `json:"seg"`
}

// Segment represents a single segment of the WLED light.
type Segment struct {
	Fx  int      `json:"fx"`
	Sx  int      `json:"sx"`
	Ix  int      `json:"ix"`
	Col [][3]int `json:"col"`
}

// Event represents a state change event.
type Event struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	On        bool   `json:"on"`
	Summary   string `json:"summary"`
}

// Server is the WLED mock server.
type Server struct {
	mu           sync.RWMutex
	state        WLEDState
	eventBuffer  [50]Event
	eventCount   int
	requestCount int
	lastUpdated  time.Time
	effectNames  map[int]string
	version      string
	commit       string
	date         string
}

// NewServer creates a new WLED mock server with default state.
func NewServer(version, commit, date string) *Server {
	s := &Server{
		version: version,
		commit:  commit,
		date:    date,
		state: WLEDState{
			On:  false,
			Bri: 128,
			Seg: []Segment{
				{Fx: 0, Sx: 128, Ix: 255, Col: [][3]int{{255, 255, 255}, {0, 255, 0}}},
			},
		},
		effectNames: profile.ReverseFxMap(),
	}
	s.addEvent("initial", s.state)
	return s
}

func (s *Server) addEvent(action string, state WLEDState) {
	var summary string
	if state.On {
		effectName := "Solid"
		if len(state.Seg) > 0 {
			if name, ok := s.effectNames[state.Seg[0].Fx]; ok {
				effectName = name
			}
		}
		summary = fmt.Sprintf("ON - %s, bri=%d (%d seg)", effectName, state.Bri, len(state.Seg))
	} else {
		summary = stateOff
	}

	s.eventBuffer[s.eventCount%50] = Event{
		Timestamp: time.Now().Format("15:04:05"),
		Action:    action,
		On:        state.On,
		Summary:   summary,
	}
	s.eventCount++
}

const stateOff = "OFF"

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

// Validation limits.
const (
	maxSegments     = 32
	maxColorsPerSeg = 64
	maxBodySize     = 1 << 20
)

func validateSegment(i int, seg Segment) error {
	if seg.Fx < 0 || seg.Fx > 255 {
		return fmt.Errorf("segment %d: fx out of range 0-255", i)
	}
	if seg.Sx < 0 || seg.Sx > 255 {
		return fmt.Errorf("segment %d: sx out of range 0-255", i)
	}
	if seg.Ix < 0 || seg.Ix > 255 {
		return fmt.Errorf("segment %d: ix out of range 0-255", i)
	}
	if len(seg.Col) > maxColorsPerSeg {
		return fmt.Errorf("segment %d: too many colors (max 64)", i)
	}
	for j, col := range seg.Col {
		for k, v := range col {
			if v < 0 || v > 255 {
				return fmt.Errorf("segment %d: color %d channel %d out of range 0-255", i, j, k)
			}
		}
	}
	return nil
}

func validateWLEDState(state WLEDState) error {
	if state.Bri < 0 || state.Bri > 255 {
		return errors.New("bri out of range 0-255")
	}
	if len(state.Seg) > maxSegments {
		return errors.New("too many segments (max 32)")
	}
	for i, seg := range state.Seg {
		if err := validateSegment(i, seg); err != nil {
			return err
		}
	}
	return nil
}

// Handler returns the HTTP handler for the mock server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/json/state", s.handleState)
	mux.HandleFunc("/json/state/events", s.handleStateEvents)
	mux.HandleFunc("/api/state", s.handleAPIState)
	mux.HandleFunc("/api/reset", s.handleAPIReset)
	mux.HandleFunc("/healthz", s.handleHealthz)
	return mux
}

// Run starts the mock server on the given port.
func (s *Server) Run(port string) {
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      s.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	slog.Info("WLED mock server starting", "port", port)
	log.Fatal(srv.ListenAndServe())
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		var newState WLEDState
		if err := json.Unmarshal(body, &newState); err != nil {
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		if err := validateWLEDState(newState); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		s.state = newState
		s.requestCount++
		s.lastUpdated = time.Now()
		s.addEvent("POST", newState)
		s.mu.Unlock()

		s.printStateColors(newState)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newState)

	case http.MethodGet:
		s.mu.RLock()
		state := s.state
		s.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(state)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleStateEvents(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	state := s.state
	count := s.eventCount
	total := 50
	if count < total {
		total = count
	}
	events := make([]Event, total)
	start := 0
	if count > 50 {
		start = count % 50
	}
	for i := 0; i < total; i++ {
		events[i] = s.eventBuffer[(start+i)%50]
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"state":  state,
		"events": events,
	})
}

type apiStateResponse struct {
	On           bool      `json:"on"`
	Seg          []Segment `json:"seg"`
	RequestCount int       `json:"request_count"`
	LastUpdated  string    `json:"last_updated"`
}

func (s *Server) handleAPIState(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	resp := apiStateResponse{
		On:           s.state.On,
		Seg:          s.state.Seg,
		RequestCount: s.requestCount,
	}
	if !s.lastUpdated.IsZero() {
		resp.LastUpdated = s.lastUpdated.Format(time.RFC3339)
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleAPIReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	s.state = WLEDState{
		On:  false,
		Bri: 128,
		Seg: []Segment{
			{Fx: 0, Sx: 128, Ix: 255, Col: [][3]int{{255, 255, 255}, {0, 255, 0}}},
		},
	}
	s.requestCount = 0
	s.lastUpdated = time.Time{}
	s.eventCount = 0
	s.addEvent("reset", s.state)
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) printStateColors(state WLEDState) {
	onOff := stateOff
	if state.On {
		onOff = "ON"
	}
	fmt.Printf("Light is %s, Brightness: %d\n", onOff, state.Bri)
	for i, seg := range state.Seg {
		effectName := s.effectNames[seg.Fx]
		fmt.Printf("Segment %d — Effect: %s, Speed: %d, Intensity: %d\n", i, effectName, seg.Sx, seg.Ix)
		for _, col := range seg.Col {
			fmt.Printf("  RGB(%d, %d, %d)\n", col[0], col[1], col[2])
		}
	}
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	s.mu.RLock()
	state := s.state
	s.mu.RUnlock()

	fxNamesJSON, _ := json.Marshal(s.effectNames)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, generateDashboardHTML(state, s.effectNames, string(fxNamesJSON), s.version, s.commit, s.date))
}

func generateDashboardHTML(state WLEDState, effectNames map[int]string, fxNamesJSON, version, commit, date string) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html lang="en" data-theme="dark"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>HOMERUN² WLED Mock</title>
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
  .main-content { flex: 1; }
  .status {
    margin-top: 20px;
    text-align: center;
    font-size: 22px;
    font-weight: bold;
    margin-bottom: 20px;
    transition: color 0.4s ease, text-shadow 0.4s ease;
  }
  .status.on {
    color: #00e676;
    text-shadow: 0 0 16px #00e676, 0 0 32px #00e67644;
  }
  .status.off { color: #64748b; text-shadow: none; }
  .container {
    display: flex;
    flex-wrap: wrap;
    gap: 20px;
    justify-content: center;
    padding: 20px;
  }
  .segment {
    background-color: #1e293b;
    border-radius: 10px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
    padding: 16px;
    width: 260px;
    border: 1px solid #334155;
    transition: filter 0.4s ease, opacity 0.4s ease;
  }
  .segment.dimmed { filter: grayscale(1); opacity: 0.5; }
  .effect-name {
    font-size: 17px;
    font-weight: bold;
    margin-bottom: 8px;
    text-align: center;
    color: #fff;
  }
  .info {
    font-size: 13px;
    margin-bottom: 10px;
    color: #818cf8;
    text-align: center;
  }
  .block {
    width: 100%;
    height: 50px;
    margin: 6px 0;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
    font-weight: bold;
    font-size: 13px;
    text-shadow: 1px 1px 2px #000;
    transition: background-color 0.4s ease, box-shadow 0.4s ease, filter 0.4s ease, opacity 0.4s ease;
  }
  .block.glow { box-shadow: 0 0 12px var(--glow-color), 0 0 24px var(--glow-color); }
  .block.dim { filter: grayscale(1); opacity: 0.35; box-shadow: none; }
  .timeline-section {
    max-width: 600px;
    margin: 24px auto;
    padding: 0 20px 20px;
  }
  .timeline-title {
    font-size: 16px;
    font-weight: bold;
    color: #818cf8;
    margin-bottom: 10px;
    text-transform: uppercase;
    letter-spacing: 1px;
  }
  #timeline {
    background-color: #1e293b;
    border-radius: 8px;
    border: 1px solid #334155;
    padding: 12px;
    max-height: 260px;
    overflow-y: auto;
    font-family: 'Courier New', monospace;
    font-size: 13px;
  }
  #timeline::-webkit-scrollbar { width: 6px; }
  #timeline::-webkit-scrollbar-track { background: #1e293b; border-radius: 3px; }
  #timeline::-webkit-scrollbar-thumb { background: #334155; border-radius: 3px; }
  .event-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 0;
    border-bottom: 1px solid #334155;
  }
  .event-row:last-child { border-bottom: none; }
  .event-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .event-dot.on { background-color: #00e676; box-shadow: 0 0 6px #00e676; }
  .event-dot.off { background-color: #64748b; }
  .event-time { color: #64748b; }
  .event-action { color: #818cf8; width: 56px; }
  .event-summary { color: #e2e8f0; }
  .build-footer { background: #1e293b; color: #475569; padding: 0.6rem 1.5rem; display: flex; gap: 1.5rem; font-size: 0.75rem; border-top: 1px solid #334155; }
  .build-footer .label { color: #64748b; }
  .build-footer .value { color: #818cf8; }
</style>
</head><body>
<div class="header-bar">
  <div>
    <h1>HOMERUN²</h1>
    <div class="subtitle">wled mock dashboard</div>
  </div>
  <div class="actions">
    <a href="/api/state">API State</a>
    <a href="/api/reset" onclick="fetch('/api/reset',{method:'POST'});setTimeout(function(){location.reload()},300);return false;">Reset</a>
  </div>
</div>
<div class="main-content">`)

	statusClass := "off"
	statusText := stateOff
	if state.On {
		statusClass = "on"
		statusText = "ON"
	}
	fmt.Fprintf(&sb, `<div id="status" class="status %s">Light is: %s</div>`, statusClass, statusText)

	sb.WriteString(`<div id="segments-container" class="container">`)
	for i, seg := range state.Seg {
		effectName := effectNames[seg.Fx]
		dimmedClass := ""
		if !state.On {
			dimmedClass = " dimmed"
		}
		fmt.Fprintf(&sb, `<div class="segment%s" data-seg="%d">`, dimmedClass, i)
		fmt.Fprintf(&sb, `<div class="effect-name">Segment %d - %s</div>`, i, effectName)
		fmt.Fprintf(&sb, `<div class="info">Speed: %d | Intensity: %d</div>`, seg.Sx, seg.Ix)
		for _, col := range seg.Col {
			color := fmt.Sprintf("rgb(%d, %d, %d)", col[0], col[1], col[2])
			glowColor := fmt.Sprintf("rgba(%d, %d, %d, 0.6)", col[0], col[1], col[2])
			blockClass := "dim"
			if state.On {
				blockClass = "glow"
			}
			fmt.Fprintf(&sb, `<div class="block %s" style="background-color: %s; --glow-color: %s;">%s</div>`,
				blockClass, color, glowColor, color)
		}
		sb.WriteString(`</div>`)
	}
	sb.WriteString("</div>")

	sb.WriteString(`<div class="timeline-section"><div class="timeline-title">Event Timeline</div><div id="timeline"></div></div>
</div>`)

	fmt.Fprintf(&sb, `
<script>
var fxNames = %s;

function updateDashboard() {
  fetch('/json/state/events')
    .then(function(r) { return r.json(); })
    .then(function(data) {
      var state = data.state;
      var events = data.events;

      var statusEl = document.getElementById('status');
      if (state.on) {
        statusEl.className = 'status on';
        statusEl.textContent = 'Light is: ON';
      } else {
        statusEl.className = 'status off';
        statusEl.textContent = 'Light is: OFF';
      }

      var container = document.getElementById('segments-container');
      container.innerHTML = '';
      for (var i = 0; i < state.seg.length; i++) {
        var seg = state.seg[i];
        var div = document.createElement('div');
        div.className = 'segment' + (state.on ? '' : ' dimmed');
        var efName = fxNames[seg.fx] || ('Effect ' + seg.fx);
        var html = '<div class="effect-name">Segment ' + i + ' - ' + efName + '</div>';
        html += '<div class="info">Speed: ' + seg.sx + ' | Intensity: ' + seg.ix + '</div>';
        for (var c = 0; c < seg.col.length; c++) {
          var col = seg.col[c];
          var rgb = 'rgb(' + col[0] + ', ' + col[1] + ', ' + col[2] + ')';
          var glow = 'rgba(' + col[0] + ', ' + col[1] + ', ' + col[2] + ', 0.6)';
          var cls = state.on ? 'glow' : 'dim';
          html += '<div class="block ' + cls + '" style="background-color: ' + rgb + '; --glow-color: ' + glow + ';">' + rgb + '</div>';
        }
        div.innerHTML = html;
        container.appendChild(div);
      }

      var tl = document.getElementById('timeline');
      var tlHtml = '';
      for (var e = events.length - 1; e >= 0; e--) {
        var ev = events[e];
        var dotClass = ev.on ? 'on' : 'off';
        tlHtml += '<div class="event-row">';
        tlHtml += '<span class="event-dot ' + dotClass + '"></span>';
        tlHtml += '<span class="event-time">' + ev.timestamp + '</span>';
        tlHtml += '<span class="event-action">' + ev.action + '</span>';
        tlHtml += '<span class="event-summary">' + ev.summary + '</span>';
        tlHtml += '</div>';
      }
      tl.innerHTML = tlHtml;
    })
    .catch(function(err) { console.error('Error fetching state:', err); });
}

updateDashboard();
setInterval(updateDashboard, 2000);
</script>
<div class="build-footer">
  <div style="display:flex;gap:1.5rem">
    <div><span class="label">version</span> <span class="value">%s</span></div>
    <div><span class="label">commit</span> <span class="value">%s</span></div>
    <div><span class="label">built</span> <span class="value">%s</span></div>
  </div>
  <div style="margin-left:auto;display:flex;align-items:center;gap:0.5rem"><span class="label">a</span> <a href="https://github.com/stuttgart-things" target="_blank" style="color:#818cf8;text-decoration:none">stuttgart-things</a> <span class="label">project</span> <img src="https://raw.githubusercontent.com/stuttgart-things/docs/main/hugo/sthings-logo.png" alt="sthings" style="height:24px;"></div>
</div>
</body></html>`, fxNamesJSON, version, shortCommit(commit), date)

	return sb.String()
}
