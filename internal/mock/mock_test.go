package mock

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleState_POST(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	body := `{"on":true,"bri":200,"seg":[{"fx":13,"sx":128,"ix":255,"col":[[255,0,0]]}]}`
	req := httptest.NewRequest(http.MethodPost, "/json/state", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var state WLEDState
	json.NewDecoder(w.Body).Decode(&state)

	if !state.On {
		t.Error("expected on=true")
	}
	if len(state.Seg) != 1 || state.Seg[0].Fx != 13 {
		t.Error("expected 1 segment with fx=13")
	}
}

func TestHandleState_GET(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/json/state", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var state WLEDState
	json.NewDecoder(w.Body).Decode(&state)
	if state.On {
		t.Error("expected initial state to be off")
	}
}

func TestHandleState_InvalidJSON(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	req := httptest.NewRequest(http.MethodPost, "/json/state", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleState_ValidationError(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	body := `{"on":true,"bri":300}`
	req := httptest.NewRequest(http.MethodPost, "/json/state", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bri=300, got %d", w.Code)
	}
}

func TestHandleAPIReset(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	// Set some state first
	body := `{"on":true,"bri":200,"seg":[{"fx":7,"sx":100,"ix":200,"col":[[0,255,0]]}]}`
	req := httptest.NewRequest(http.MethodPost, "/json/state", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Reset
	req = httptest.NewRequest(http.MethodPost, "/api/reset", http.NoBody)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify reset state
	req = httptest.NewRequest(http.MethodGet, "/json/state", http.NoBody)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var state WLEDState
	json.NewDecoder(w.Body).Decode(&state)
	if state.On {
		t.Error("expected state to be off after reset")
	}
}

func TestHandleHealthz(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	b, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(b), `"status":"ok"`) {
		t.Errorf("unexpected body: %s", string(b))
	}
}

func TestHandleAPIState(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	handler := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/state", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp apiStateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.RequestCount != 0 {
		t.Errorf("expected 0 requests, got %d", resp.RequestCount)
	}
}

func TestValidateSegment_OutOfRange(t *testing.T) {
	seg := Segment{Fx: 256, Sx: 0, Ix: 0, Col: nil}
	if err := validateSegment(0, seg); err == nil {
		t.Error("expected error for fx=256")
	}
}

func TestValidateWLEDState_TooManySegments(t *testing.T) {
	segs := make([]Segment, 33)
	state := WLEDState{On: true, Bri: 128, Seg: segs}
	if err := validateWLEDState(state); err == nil {
		t.Error("expected error for 33 segments")
	}
}
