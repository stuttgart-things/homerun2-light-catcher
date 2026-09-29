package mock

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func post(t *testing.T, h http.Handler, body string) WLEDState {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/json/state", strings.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("POST %s: %d %s", body, w.Code, w.Body.String())
	}
	var st WLEDState
	if err := json.NewDecoder(w.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	return st
}

func TestHandlePalettes(t *testing.T) {
	w := httptest.NewRecorder()
	NewServer("test", "abc1234", "2026-01-01").Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/json/pal", http.NoBody))
	var names []string
	if err := json.NewDecoder(w.Body).Decode(&names); err != nil {
		t.Fatal(err)
	}
	if len(names) < 14 || names[8] != "Lava" || names[13] != "Sunset" {
		t.Fatalf("want WLED's numbering (Lava 8, Sunset 13), got %v", names)
	}
}

// Like WLED, a POST changes only what it carries.
func TestMergeState(t *testing.T) {
	h := NewServer("test", "abc1234", "2026-01-01").Handler()
	post(t, h, `{"on":true,"bri":120,"seg":[{"fx":9,"pal":11,"col":[[1,2,3]]}]}`)

	st := post(t, h, `{"on":false}`)
	if st.On || st.Bri != 120 || len(st.Seg) != 1 || st.Seg[0].Fx != 9 {
		t.Fatalf("off must keep bri and segments, got %+v", st)
	}

	st = post(t, h, `{"seg":[{"pal":8}]}`)
	if st.Seg[0].Pal != 8 || st.Seg[0].Col[0] != [3]int{1, 2, 3} || st.Seg[0].Fx != 9 {
		t.Fatalf("palette-only must keep fx and colors, got %+v", st.Seg[0])
	}

	st = post(t, h, `{"seg":[{"id":2,"fx":38}]}`)
	if len(st.Seg) != 3 || st.Seg[2].Fx != 38 || st.Seg[0].Fx != 9 {
		t.Fatalf("id 2 must create segments up to 2 and leave 0 alone, got %+v", st.Seg)
	}
}
