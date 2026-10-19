package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMount(t *testing.T) {
	const (
		get, post = http.MethodGet, http.MethodPost
		light     = "/light"
		ui        = light + "/"
		ok        = "health"
	)
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("app " + r.Method + " " + r.URL.Path))
	})
	health := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(ok))
	})

	cases := []struct {
		basePath, method, path string
		code                   int
		body, location         string
	}{
		{"", get, "/", 200, "app GET /", ""},
		{"", get, "/timeline", 200, "app GET /timeline", ""},
		{"", get, "/health", 200, ok, ""},
		{"", get, "/healthz", 200, ok, ""},
		{light, get, ui, 200, "app GET /", ""},
		{light, get, "/light/timeline", 200, "app GET /timeline", ""},
		{light, post, "/light/api/events/3/replay", 200, "app POST /api/events/3/replay", ""},
		{light, get, "/health", 200, ok, ""},
		{light, get, "/healthz", 200, ok, ""},
		{light, get, "/light/health", 200, "app GET /health", ""},
		{light, get, "/light", 307, "", ui},
		{light, get, "/", 307, "", ui},
		{light, get, "/timeline", 404, "", ""},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		Mount(tc.basePath, app, health).ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, http.NoBody))
		if rec.Code != tc.code {
			t.Errorf("base %q %s %s: code %d, want %d", tc.basePath, tc.method, tc.path, rec.Code, tc.code)
			continue
		}
		if tc.body != "" && rec.Body.String() != tc.body {
			t.Errorf("base %q %s %s: body %q, want %q", tc.basePath, tc.method, tc.path, rec.Body.String(), tc.body)
		}
		if loc := rec.Header().Get("Location"); tc.location != "" && loc != tc.location {
			t.Errorf("base %q %s %s: Location %q, want %q", tc.basePath, tc.method, tc.path, loc, tc.location)
		}
	}
}
