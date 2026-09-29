package mock

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// handlePalettes serves the palette list like WLED's GET /json/pal.
func (s *Server) handlePalettes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.palettes)
}

// snapshot returns a copy of the current state.
func (s *Server) snapshot() WLEDState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := s.state
	st.Seg = append([]Segment(nil), s.state.Seg...)
	return st
}

// mergeState applies a POSTed state the way WLED does: only the fields the
// request carries change. A segment with an id updates that segment (created
// if needed), one without updates the segment at its position; within a
// segment, absent fields keep their value -- so a palette-only update keeps
// the colors, and writing back a state read earlier restores it. Before this
// the mock REPLACED the whole state, so {"on":false} also wiped the segments
// and "restore" could not be shown against it (#77).
func mergeState(cur WLEDState, body []byte, incoming WLEDState) (WLEDState, error) {
	var raw struct {
		On  json.RawMessage   `json:"on"`
		Bri json.RawMessage   `json:"bri"`
		Seg []json.RawMessage `json:"seg"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return cur, errors.New("invalid JSON format")
	}
	if raw.On != nil {
		cur.On = incoming.On
	}
	if raw.Bri != nil {
		cur.Bri = incoming.Bri
	}
	for i, rs := range raw.Seg {
		var present map[string]json.RawMessage
		if err := json.Unmarshal(rs, &present); err != nil {
			return cur, fmt.Errorf("segment %d: invalid JSON", i)
		}
		in := incoming.Seg[i]
		idx := i
		if in.ID != nil {
			idx = *in.ID
		}
		for len(cur.Seg) <= idx {
			n := len(cur.Seg)
			cur.Seg = append(cur.Seg, Segment{ID: &n, Sx: 128, Ix: 128, Col: [][3]int{{255, 255, 255}}})
		}
		seg := cur.Seg[idx]
		id := idx
		seg.ID = &id
		if _, ok := present["fx"]; ok {
			seg.Fx = in.Fx
		}
		if _, ok := present["sx"]; ok {
			seg.Sx = in.Sx
		}
		if _, ok := present["ix"]; ok {
			seg.Ix = in.Ix
		}
		if _, ok := present["pal"]; ok {
			seg.Pal = in.Pal
		}
		if _, ok := present["col"]; ok {
			seg.Col = in.Col
		}
		cur.Seg[idx] = seg
	}
	return cur, nil
}
