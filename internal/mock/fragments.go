package mock

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"
)

// Micro-frontend fragments for homerun2-portal, as internal/dashboard offers
// them for the light-catcher. The portal reads <base>/ui/manifest.json and
// loads each <base>/ui/fragment/<id> with htmx into its overview. A fragment
// is bare HTML styled only by the portal's hr-* classes (static/homerun2.css
// in homerun2-portal); every URL in it carries the base path, because it is
// served on the portal's origin.

const (
	stateFragmentRefresh = 3
	// stateFragmentSegments caps the segments listed; a strip can have 32.
	stateFragmentSegments = 8
)

// Fragment is one entry of the manifest.
type Fragment struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	URL            string `json:"url"`
	RefreshSeconds int    `json:"refreshSeconds"`
}

// Manifest lists the fragments the mock offers to the portal.
type Manifest struct {
	Name      string     `json:"name"`
	Title     string     `json:"title"`
	Version   string     `json:"version"`
	BasePath  string     `json:"basePath"`
	Fragments []Fragment `json:"fragments"`
}

// handleManifest serves GET /ui/manifest.json.
func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Manifest{
		Name:     "homerun2-wled-mock",
		Title:    "WLED Mock",
		Version:  s.version,
		BasePath: s.basePath,
		Fragments: []Fragment{{
			ID:             "state",
			Title:          "Strip state",
			URL:            s.basePath + "/ui/fragment/state",
			RefreshSeconds: stateFragmentRefresh,
		}},
	})
}

// handleStateFragment serves GET /ui/fragment/state.
func (s *Server) handleStateFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, s.renderStateFragment())
}

// renderStateFragment renders the strip as an hr-list: on/off and brightness
// with the time of the last change, then one item per segment with its first
// colour, effect and palette. Names are escaped: the effect list can come
// from a file (WLED_EFFECTS_FILE), and the portal shows the fragment to every
// viewer.
func (s *Server) renderStateFragment() string {
	esc := html.EscapeString
	s.mu.RLock()
	state := s.state
	state.Seg = append([]Segment(nil), s.state.Seg...)
	updated := s.lastUpdated
	s.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString(`<div class="hr-frag"><ul class="hr-list"><li class="hr-item">`)
	if !updated.IsZero() {
		fmt.Fprintf(&sb, `<span class="hr-time">%s</span>`, updated.Format(time.TimeOnly))
	}
	if !state.On {
		sb.WriteString(`<span class="hr-dot"></span><span class="hr-title hr-muted">Light is off</span></li>`)
	} else {
		fmt.Fprintf(&sb, `<span class="hr-title">On</span><span class="hr-muted">brightness %d/255 (%d%%)</span></li>`,
			state.Bri, (state.Bri*100+127)/255)
		for i, seg := range state.Seg {
			if i == stateFragmentSegments {
				fmt.Fprintf(&sb, `<li class="hr-item hr-muted">+%d more segments</li>`, len(state.Seg)-i)
				break
			}
			sb.WriteString(`<li class="hr-item">`)
			fmt.Fprintf(&sb, `<span class="hr-dot"%s></span>`, swatchStyle(seg.Col))
			fmt.Fprintf(&sb, `<span class="hr-title">%s</span>`, esc(s.effectName(seg.Fx)))
			if pal := s.paletteName(seg.Pal); pal != "" {
				fmt.Fprintf(&sb, `<span class="hr-system">%s</span>`, esc(pal))
			}
			sb.WriteString(`</li>`)
		}
	}
	fmt.Fprintf(&sb, `</ul><a class="hr-frag-more" href="%s/">Open simulator →</a></div>`, esc(s.basePath))
	return sb.String()
}

// effectName names an effect ID like the dashboard does.
func (s *Server) effectName(fx int) string {
	if name, ok := s.effectNames[fx]; ok {
		return name
	}
	return fmt.Sprintf("Effect %d", fx)
}

// paletteName names a palette ID; "" for palette 0 (WLED's "Default", i.e.
// none set) and IDs the list does not know.
func (s *Server) paletteName(pal int) string {
	if pal <= 0 || pal >= len(s.palettes) {
		return ""
	}
	return s.palettes[pal]
}

// swatchStyle colours the dot with the segment's first colour.
func swatchStyle(col [][3]int) string {
	if len(col) == 0 {
		return ""
	}
	c := col[0]
	return fmt.Sprintf(` style="background: rgb(%d, %d, %d)"`, c[0], c[1], c[2])
}
