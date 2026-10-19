package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMount(t *testing.T) {
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("app " + r.Method + " " + r.URL.Path))
	})
	health := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("health"))
	})

	cases := []struct {
		basePath, method, path string
		code                   int
		body, location         string
	}{
		{"", "GET", "/", 200, "app GET /", ""},
		{"", "GET", "/timeline", 200, "app GET /timeline", ""},
		{"", "GET", "/health", 200, "health", ""},
		{"", "GET", "/healthz", 200, "health", ""},
		{"/light", "GET", "/light/", 200, "app GET /", ""},
		{"/light", "GET", "/light/timeline", 200, "app GET /timeline", ""},
		{"/light", "POST", "/light/api/events/3/replay", 200, "app POST /api/events/3/replay", ""},
		{"/light", "GET", "/health", 200, "health", ""},
		{"/light", "GET", "/healthz", 200, "health", ""},
		{"/light", "GET", "/light/health", 200, "app GET /health", ""},
		{"/light", "GET", "/light", 307, "", "/light/"},
		{"/light", "GET", "/", 307, "", "/light/"},
		{"/light", "GET", "/timeline", 404, "", ""},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		Mount(tc.basePath, app, health).ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
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
