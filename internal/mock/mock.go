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
}

// NewServer creates a new WLED mock server with default state.
func NewServer() *Server {
	s := &Server{
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
		summary = "OFF"
	}

	s.eventBuffer[s.eventCount%50] = Event{
		Timestamp: time.Now().Format("15:04:05"),
		Action:    action,
		On:        state.On,
		Summary:   summary,
	}
	s.eventCount++
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
	onOff := "OFF"
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
	fmt.Fprint(w, generateDashboardHTML(state, s.effectNames, string(fxNamesJSON)))
}

func generateDashboardHTML(state WLEDState, effectNames map[int]string, fxNamesJSON string) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>WLED Mock Dashboard</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: 'Segoe UI', 'Roboto', Arial, sans-serif;
    background-color: #1a1a2e;
    color: #e0e0e0;
    min-height: 100vh;
  }
  h1 {
    text-align: center;
    padding: 24px 0 8px;
    font-size: 28px;
    letter-spacing: 1px;
    color: #fff;
  }
  #endpoint {
    text-align: center;
    color: #888;
    font-size: 14px;
    margin-bottom: 16px;
    font-family: 'Courier New', monospace;
  }
  .status {
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
  .status.off { color: #666; text-shadow: none; }
  .container {
    display: flex;
    flex-wrap: wrap;
    gap: 20px;
    justify-content: center;
    padding: 20px;
  }
  .segment {
    background-color: #16213e;
    border-radius: 10px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
    padding: 16px;
    width: 260px;
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
    color: #aaa;
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
    padding: 0 20px;
  }
  .timeline-title {
    font-size: 16px;
    font-weight: bold;
    color: #aaa;
    margin-bottom: 10px;
    text-transform: uppercase;
    letter-spacing: 1px;
  }
  #timeline {
    background-color: #0f0f23;
    border-radius: 8px;
    padding: 12px;
    max-height: 260px;
    overflow-y: auto;
    font-family: 'Courier New', monospace;
    font-size: 13px;
  }
  #timeline::-webkit-scrollbar { width: 6px; }
  #timeline::-webkit-scrollbar-track { background: #0f0f23; border-radius: 3px; }
  #timeline::-webkit-scrollbar-thumb { background: #333; border-radius: 3px; }
  .event-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 0;
    border-bottom: 1px solid #1a1a2e;
  }
  .event-row:last-child { border-bottom: none; }
  .event-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .event-dot.on { background-color: #00e676; box-shadow: 0 0 6px #00e676; }
  .event-dot.off { background-color: #555; }
  .event-time { color: #666; }
  .event-action { color: #888; width: 56px; }
  .event-summary { color: #ccc; }
</style>
</head><body>
<h1>WLED Mock Dashboard</h1>
<div id="endpoint"></div>`)

	statusClass := "off"
	statusText := "OFF"
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

	sb.WriteString(`<div class="timeline-section"><div class="timeline-title">Event Timeline</div><div id="timeline"></div></div>`)

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

document.getElementById('endpoint').textContent = window.location.origin + '/json/state';
updateDashboard();
setInterval(updateDashboard, 2000);
</script>
</body></html>`, fxNamesJSON)

	return sb.String()
}
