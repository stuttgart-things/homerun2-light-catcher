package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(t *testing.T, h http.Handler, path string) (code int, body, location string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, http.NoBody))
	b, _ := io.ReadAll(rec.Body)
	return rec.Code, string(b), rec.Header().Get("Location")
}

func TestMountUnderBasePath(t *testing.T) {
	app := http.NewServeMux()
	app.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "app:"+r.URL.Path) })
	health := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "healthy") })
	h := Mount("/light", app, health)

	for path, want := range map[string]string{
		"/light/":         "app:/",
		"/light/timeline": "app:/timeline",
		"/health":         "healthy", // probes and the portal, at the root
		"/healthz":        "healthy",
	} {
		if code, body, _ := serve(t, h, path); code != http.StatusOK || body != want {
			t.Errorf("GET %s = %d %q, want %q", path, code, body, want)
		}
	}
	for _, path := range []string{"/", "/light"} {
		if code, _, loc := serve(t, h, path); code/100 != 3 || loc != "/light/" {
			t.Errorf("GET %s = %d → %q, want a redirect to /light/", path, code, loc)
		}
	}
	if code, _, _ := serve(t, h, "/timeline"); code != http.StatusNotFound {
		t.Errorf("GET /timeline without the prefix = %d, want 404", code)
	}
}

func TestMountWithoutBasePathServesAppAsIs(t *testing.T) {
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "app:"+r.URL.Path) })
	h := Mount("", app, http.NotFoundHandler())
	if code, body, _ := serve(t, h, "/timeline"); code != http.StatusOK || body != "app:/timeline" {
		t.Errorf("GET /timeline = %d %q", code, body)
	}
}
