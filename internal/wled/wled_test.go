package wled

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendEffect(t *testing.T) {
	var received map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/state" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := SendEffect(srv.URL, 13, [][3]int{{255, 0, 0}, {0, 255, 0}}, EffectMeta{})
	if err != nil {
		t.Fatalf("SendEffect: %v", err)
	}

	if received["on"] != true {
		t.Error("expected on=true")
	}

	seg, ok := received["seg"].([]any)
	if !ok || len(seg) != 1 {
		t.Fatal("expected 1 segment")
	}

	s := seg[0].(map[string]any)
	if s["fx"].(float64) != 13 {
		t.Errorf("expected fx=13, got %v", s["fx"])
	}
}

func TestTurnOff(t *testing.T) {
	var received map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := TurnOff(srv.URL)
	if err != nil {
		t.Fatalf("TurnOff: %v", err)
	}

	if received["on"] != false {
		t.Error("expected on=false")
	}
}

func TestSendEffect_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := SendEffect(srv.URL, 0, [][3]int{{255, 255, 255}}, EffectMeta{})
	if err == nil {
		t.Error("expected error for 500 response")
	}
}
