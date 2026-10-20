package mock

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// WLEDState represents the state of the WLED light.
type WLEDState struct {
	On  bool      `json:"on"`
	Bri int       `json:"bri"`
	Seg []Segment `json:"seg"`
}

// Segment represents a single segment of the WLED light.
type Segment struct {
	// ID addresses the segment, as on WLED; state returned by GET carries it.
	ID  *int     `json:"id,omitempty"`
	Fx  int      `json:"fx"`
	Sx  int      `json:"sx"`
	Ix  int      `json:"ix"`
	Pal int      `json:"pal"`
	Col [][3]int `json:"col"`
}

// Event represents a state change event.
type Event struct {
	Timestamp string   `json:"timestamp"`
	Action    string   `json:"action"`
	On        bool     `json:"on"`
	Summary   string   `json:"summary"`
	Severity  string   `json:"severity,omitempty"`
	System    string   `json:"system,omitempty"`
	Effect    string   `json:"effect,omitempty"`
	Color     string   `json:"color,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// eventMeta holds metadata from the light-catcher payload.
type eventMeta struct {
	Severity string   `json:"_severity"`
	System   string   `json:"_system"`
	Effect   string   `json:"_effect"`
	Color    string   `json:"_color"`
	Tags     []string `json:"_tags"`
}

// Server is the WLED mock server.
type Server struct {
	mu           sync.RWMutex
	state        WLEDState
	eventBuffer  [50]Event
	eventCount   int
	requestCount int
	lastUpdated  time.Time
	// effects is served on /json/eff; index = effect ID, as on a device.
	effects []string
	// palettes is served on /json/pal; index = palette ID.
	palettes    []string
	effectNames map[int]string
	version     string
	commit      string
	date        string
	// basePath prefixes the dashboard and every URL it hands out (BASE_PATH,
	// e.g. "/wled" behind homerun2-portal). Empty serves at the root.
	basePath string
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
	}
	effects, err := loadEffects()
	if err != nil {
		slog.Error("falling back to the built-in WLED 16 effect list", "error", err)
		effects = wled16Effects
	}
	s.effects = effects
	palettes, err := loadPalettes()
	if err != nil {
		slog.Error("falling back to the built-in WLED palette list", "error", err)
		palettes = wledPalettes
	}
	s.palettes = palettes
	s.effectNames = effectNamesByID(effects)
	s.addEvent("initial", s.state, nil)
	return s
}

func (s *Server) addEvent(action string, state WLEDState, meta *eventMeta) {
	var summary string
	if state.On {
		effectName := fxSolid
		if len(state.Seg) > 0 {
			if name, ok := s.effectNames[state.Seg[0].Fx]; ok {
				effectName = name
			}
		}
		summary = fmt.Sprintf("ON - %s, bri=%d (%d seg)", effectName, state.Bri, len(state.Seg))
	} else {
		summary = stateOff
	}

	ev := Event{
		Timestamp: time.Now().Format("15:04:05"),
		Action:    action,
		On:        state.On,
		Summary:   summary,
	}
	if meta != nil {
		ev.Severity = meta.Severity
		ev.System = meta.System
		ev.Effect = meta.Effect
		ev.Color = meta.Color
		ev.Tags = meta.Tags
	}
	s.eventBuffer[s.eventCount%50] = ev
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
	for _, f := range []struct {
		name string
		v    int
	}{{"fx", seg.Fx}, {"sx", seg.Sx}, {"ix", seg.Ix}, {"pal", seg.Pal}} {
		if f.v < 0 || f.v > 255 {
			return fmt.Errorf("segment %d: %s out of range 0-255", i, f.name)
		}
	}
	if seg.ID != nil && (*seg.ID < 0 || *seg.ID >= maxSegments) {
		return fmt.Errorf("segment %d: id out of range 0-%d", i, maxSegments-1)
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

// SetBasePath sets the URL prefix the dashboard is served under, as
// config.ParseBasePath normalised it ("/wled", or "" for the root).
func (s *Server) SetBasePath(basePath string) {
	s.basePath = basePath
}

// Handler returns the HTTP handler for the mock server. With a base path, the
// whole mock answers under it (homerun2-portal forwards the path unchanged),
// / redirects there, and two things stay at the root as well: the health
// endpoints, for the probes and the portal's status check, and the WLED JSON
// API under /json/, which light-catcher profiles address like a device
// (http://homerun2-wled-mock/json/state).
func (s *Server) Handler() http.Handler {
	app := s.routes()
	if s.basePath == "" {
		return app
	}
	mux := http.NewServeMux()
	health := http.HandlerFunc(s.handleHealthz)
	mux.Handle("/health", health)
	mux.Handle("/healthz", health)
	mux.Handle("/json/", app)
	toUI := http.RedirectHandler(s.basePath+"/", http.StatusTemporaryRedirect)
	mux.Handle(s.basePath+"/", http.StripPrefix(s.basePath, app))
	mux.Handle(s.basePath, toUI)
	mux.Handle("/{$}", toUI)
	return mux
}

// routes serves the mock at the root of its own mux.
func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/json/state", s.handleState)
	mux.HandleFunc("/json/eff", s.handleEffects)
	mux.HandleFunc("/json/pal", s.handlePalettes)
	mux.HandleFunc("/json/state/events", s.handleStateEvents)
	mux.HandleFunc("/api/state", s.handleAPIState)
	mux.HandleFunc("/api/reset", s.handleAPIReset)
	mux.HandleFunc("GET /ui/manifest.json", s.handleManifest)
	mux.HandleFunc("GET /ui/fragment/state", s.handleStateFragment)
	mux.HandleFunc("/health", s.handleHealthz)
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

	slog.Info("WLED mock server starting", "port", port, "base_path", s.basePath)
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

		// Parse optional metadata from light-catcher
		var meta eventMeta
		json.Unmarshal(body, &meta)
		var metaPtr *eventMeta
		if meta.Severity != "" || meta.System != "" {
			metaPtr = &meta
		}

		merged, mergeErr := mergeState(s.snapshot(), body, newState)
		if mergeErr != nil {
			http.Error(w, mergeErr.Error(), http.StatusBadRequest)
			return
		}
		newState = merged

		s.mu.Lock()
		s.state = newState
		s.requestCount++
		s.lastUpdated = time.Now()
		s.addEvent("POST", newState, metaPtr)
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
	s.addEvent("reset", s.state, nil)
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}

// handleEffects serves the effect list like WLED's GET /json/eff, which the
// light-catcher reads to turn effect names into IDs.
func (s *Server) handleEffects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.effects)
}

// handleHealthz answers /health and /healthz. version and commit are what
// homerun2-portal shows in its status.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": s.version,
		"commit":  s.commit,
		"date":    s.date,
	})
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
	page := generateDashboardHTML(state, s.effectNames, string(fxNamesJSON), s.basePath, s.version, s.commit, s.date)
	if r.Header.Get("X-Forwarded-Prefix") != "" {
		// Through homerun2-portal: its design tokens win over the page's copy.
		page = strings.Replace(page, "</head>", portalTokensLink+"</head>", 1)
	}
	fmt.Fprint(w, page)
}

// portalTokensLink loads homerun2-portal's design tokens from the portal's
// own origin.
const portalTokensLink = `<link rel="stylesheet" href="/static/homerun2-tokens.css">`

func generateDashboardHTML(state WLEDState, effectNames map[int]string, fxNamesJSON, basePath, version, commit, date string) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html lang="en" data-theme="dark"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>HOMERUN² WLED Mock</title>
<script>
  // In homerun2-portal's iframe the portal shows the brand and the service
  // name: hr-embedded switches to the compact header. Set before the first
  // paint, so the big banner never flashes.
  if (window.self !== window.top) document.documentElement.classList.add('hr-embedded');
</script>
<link href="https://fonts.googleapis.com/css2?family=Press+Start+2P&display=swap" rel="stylesheet">
<style>
  /* The homerun2 design tokens: a copy of homerun2-portal's
     static/homerun2-tokens.css for running without the portal. Behind it
     handleDashboard also links the portal's file (after this block), and its
     :root wins over this zero-specificity copy. */
  :where(:root) {
    --hr-bg: #0b1120;
    --hr-surface: #111827;
    --hr-card: #1e293b;
    --hr-border: #334155;
    --hr-text: #e2e8f0;
    --hr-muted: #94a3b8;
    --hr-accent: #6366f1;
    --hr-accent-2: #a855f7;
    --hr-accent-3: #f97316;
    --hr-ok: #22c55e;
    --hr-info: #38bdf8;
    --hr-warning: #eab308;
    --hr-error: #ef4444;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: 'Segoe UI', 'Roboto', Arial, sans-serif;
    background-color: var(--hr-bg);
    color: var(--hr-text);
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }
  .header-bar { background: linear-gradient(135deg, var(--hr-accent) 0%, var(--hr-accent-2) 100%); color: #f8fafc; padding: 1.8rem 1.5rem; display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 3px solid var(--hr-accent-3); }
  .header-bar h1 { margin: 0; font-family: 'Press Start 2P', cursive; font-size: 2.2rem; color: #ffffff; letter-spacing: 0.08em; text-shadow: 3px 3px 0px rgba(0,0,0,0.3); }
  .header-bar .subtitle { font-family: 'Press Start 2P', cursive; font-size: 0.7rem; color: #fbbf24; margin-top: 0.5rem; letter-spacing: 0.12em; text-transform: uppercase; }
  .header-bar .actions { display: flex; gap: 0.75rem; align-items: center; }
  .header-bar .actions a { color: var(--hr-text); font-size: 0.85rem; text-decoration: none; }
  .header-bar .actions a:hover { color: #f8fafc; }
  /* The compact header inside homerun2-portal (README there, "Embedded
     header"): one line on the portal's surface, no brand, the subtitle on the
     left and the links on the right. */
  .hr-embedded .header-bar { font-family: var(--hr-font, system-ui, sans-serif); background: var(--hr-surface); border-bottom: 1px solid var(--hr-border); min-height: 44px; padding: 6px 16px; align-items: center; }
  .hr-embedded .header-bar h1 { display: none; }
  .hr-embedded .header-bar .subtitle { font-family: inherit; font-size: 13px; color: var(--hr-muted); margin: 0; letter-spacing: normal; text-transform: none; }
  .hr-embedded .header-bar .actions a { font-size: 12px; color: var(--hr-accent); text-decoration: none; }
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
    color: var(--hr-ok);
    text-shadow: 0 0 16px var(--hr-ok), 0 0 32px color-mix(in srgb, var(--hr-ok) 27%, transparent);
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
    background-color: var(--hr-card);
    border-radius: 10px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
    padding: 16px;
    width: 260px;
    border: 1px solid var(--hr-border);
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
    background-color: var(--hr-card);
    border-radius: 8px;
    border: 1px solid var(--hr-border);
    padding: 12px;
    max-height: 260px;
    overflow-y: auto;
    font-family: 'Courier New', monospace;
    font-size: 13px;
  }
  #timeline::-webkit-scrollbar { width: 6px; }
  #timeline::-webkit-scrollbar-track { background: var(--hr-card); border-radius: 3px; }
  #timeline::-webkit-scrollbar-thumb { background: var(--hr-border); border-radius: 3px; }
  .event-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 0;
    border-bottom: 1px solid var(--hr-border);
  }
  .event-row:last-child { border-bottom: none; }
  .event-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .event-dot.on { background-color: var(--hr-ok); box-shadow: 0 0 6px var(--hr-ok); }
  .event-dot.off { background-color: #64748b; }
  .event-time { color: #64748b; font-family: 'Courier New', monospace; }
  .event-action { color: #818cf8; width: 56px; }
  .event-summary { color: var(--hr-text); }
  .event-badge { font-size: 11px; font-weight: bold; padding: 2px 6px; border-radius: 4px; text-transform: uppercase; min-width: 60px; text-align: center; }
  .sev-error { background: color-mix(in srgb, var(--hr-error) 20%, transparent); color: var(--hr-error); }
  .sev-warning { background: color-mix(in srgb, var(--hr-warning) 20%, transparent); color: var(--hr-warning); }
  .sev-success { background: color-mix(in srgb, var(--hr-ok) 20%, transparent); color: var(--hr-ok); }
  .sev-info { background: color-mix(in srgb, var(--hr-info) 20%, transparent); color: var(--hr-info); }
  .event-meta { color: #818cf8; min-width: 80px; }
  .event-meta-effect { color: var(--hr-text); }
  .event-tag { font-size: 11px; color: #fbbf24; background: rgba(251,191,36,0.12); border: 1px solid rgba(251,191,36,0.35); border-radius: 4px; padding: 1px 6px; }
  .build-footer { background: var(--hr-surface); color: #475569; padding: 0.6rem 1.5rem; display: flex; gap: 1.5rem; font-size: 0.75rem; border-top: 1px solid var(--hr-border); }
  .build-footer .label { color: #64748b; }
  .build-footer .value { color: #818cf8; }
</style>
</head><body>
<div class="header-bar">
  <div>
    <h1>HOMERUN²</h1>
    <div class="subtitle">LED strip simulator</div>
  </div>
  <div class="actions">
`)
	base := html.EscapeString(basePath)
	fmt.Fprintf(&sb, `    <a href="%s/api/state">API State</a>
    <a href="%s/api/reset" onclick="fetch(this.href,{method:'POST'});setTimeout(function(){location.reload()},300);return false;">Reset</a>
`, base, base)
	sb.WriteString(`  </div>
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

	basePathJS, _ := json.Marshal(basePath)
	fmt.Fprintf(&sb, `
<script>
var fxNames = %s;
// URL prefix behind homerun2-portal (BASE_PATH), "" at the root.
var basePath = %s;

function esc(s) {
  return String(s).replace(/[&<>"']/g, function(c) {
    return {'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[c];
  });
}

function updateDashboard() {
  fetch(basePath + '/json/state/events')
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
        if (ev.severity) {
          var sevCls = ev.severity.toUpperCase() === 'ERROR' ? 'sev-error' : ev.severity.toUpperCase() === 'WARNING' ? 'sev-warning' : ev.severity.toUpperCase() === 'SUCCESS' ? 'sev-success' : 'sev-info';
          tlHtml += '<span class="event-badge ' + sevCls + '">' + ev.severity.toUpperCase() + '</span>';
          tlHtml += '<span class="event-meta">' + ev.system + '</span>';
          tlHtml += '<span class="event-meta-effect">' + ev.effect + ' / ' + ev.color + '</span>';
          if (ev.tags) {
            for (var t = 0; t < ev.tags.length; t++) {
              tlHtml += '<span class="event-tag">' + esc(ev.tags[t]) + '</span>';
            }
          }
        } else {
          tlHtml += '<span class="event-summary">' + ev.summary + '</span>';
        }
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
</body></html>`, fxNamesJSON, basePathJS, version, shortCommit(commit), date)

	return sb.String()
}
