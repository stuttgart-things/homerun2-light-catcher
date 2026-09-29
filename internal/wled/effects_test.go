package wled

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// effectServer serves list() on /json/eff and counts the requests.
func effectServer(t *testing.T, status int, list func() []string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/eff" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		hits.Add(1)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		_ = json.NewEncoder(w).Encode(list())
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func withRefetchAfter(t *testing.T, d time.Duration) {
	t.Helper()
	old := refetchAfter
	refetchAfter = d
	t.Cleanup(func() { refetchAfter = old })
}

func TestResolveEffect_NumericNeedsNoDevice(t *testing.T) {
	srv, hits := effectServer(t, http.StatusOK, func() []string { return nil })
	id, src, err := ResolveEffect(srv.URL, " 23 ")
	if err != nil || id != 23 || src != SourceNumeric {
		t.Fatalf("got %d %q %v, want 23 numeric", id, src, err)
	}
	if hits.Load() != 0 {
		t.Fatalf("a numeric fx must not query the device, got %d requests", hits.Load())
	}
	if _, _, err := ResolveEffect(srv.URL, "-1"); err == nil {
		t.Fatal("a negative ID must be refused")
	}
}

// The #77 case: the device numbers effects differently from the old table.
func TestResolveEffect_DeviceListWinsAndIgnoresCase(t *testing.T) {
	names := make([]string, 43)
	names[0], names[17], names[42] = "Solid", "Twinkle", "Fireworks"
	srv, hits := effectServer(t, http.StatusOK, func() []string { return names })

	for fx, want := range map[string]int{"Fireworks": 42, "twinkle": 17, "SOLID": 0} {
		id, src, err := ResolveEffect(srv.URL, fx)
		if err != nil || id != want || src != SourceDevice {
			t.Errorf("%s: got %d %q %v, want %d from the device", fx, id, src, err, want)
		}
	}
	if hits.Load() != 1 {
		t.Errorf("the list is cached per endpoint, got %d requests", hits.Load())
	}
}

// A device that answers is authoritative: a name it does not list is an
// error even though the fallback table knows it.
func TestResolveEffect_DeviceIsAuthoritative(t *testing.T) {
	withRefetchAfter(t, time.Hour)
	srv, _ := effectServer(t, http.StatusOK, func() []string { return []string{"Solid", "Blink"} })
	if _, _, err := ResolveEffect(srv.URL, "Aurora"); err == nil {
		t.Fatal("Aurora is not on this device; the fallback table must not answer for it")
	}
}

// An unknown name reloads the list -- a device update may have added it --
// but at most once per refetchAfter.
func TestResolveEffect_UnknownNameRefetchesThrottled(t *testing.T) {
	var updated atomic.Bool
	srv, hits := effectServer(t, http.StatusOK, func() []string {
		if updated.Load() {
			return []string{"Solid", "Strobe"}
		}
		return []string{"Solid"}
	})

	withRefetchAfter(t, time.Hour)
	if _, _, err := ResolveEffect(srv.URL, "Strobe"); err == nil {
		t.Fatal("Strobe is not on the device yet")
	}
	updated.Store(true)
	if _, _, err := ResolveEffect(srv.URL, "Strobe"); err == nil {
		t.Fatal("within refetchAfter the cached list must answer")
	}
	if hits.Load() != 1 {
		t.Fatalf("throttled: want 1 request, got %d", hits.Load())
	}

	withRefetchAfter(t, 0)
	id, src, err := ResolveEffect(srv.URL, "Strobe")
	if err != nil || id != 1 || src != SourceDevice {
		t.Fatalf("after refetchAfter the new list must be read: got %d %q %v", id, src, err)
	}
}

// Unreachable device: the fallback table answers, and the device is not asked
// again for every message.
func TestResolveEffect_UnreachableDeviceFallsBack(t *testing.T) {
	withRefetchAfter(t, time.Hour)
	srv, hits := effectServer(t, http.StatusInternalServerError, nil)

	for i := 0; i < 3; i++ {
		id, src, err := ResolveEffect(srv.URL, "fireworks")
		if err != nil || id != 42 || src != SourceFallback {
			t.Fatalf("got %d %q %v, want 42 from the fallback table", id, src, err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("a failing device is asked once per refetchAfter, got %d requests", hits.Load())
	}
	if _, _, err := ResolveEffect(srv.URL, "No Such Effect"); err == nil {
		t.Fatal("a name in neither list must be an error")
	}
}

func TestEffectIDs_SkipsReservedAndKeepsFirst(t *testing.T) {
	ids := effectIDs([]string{"Solid", "RSVD", "-", "Blink", "solid", " "})
	if len(ids) != 2 || ids["solid"] != 0 || ids["blink"] != 3 {
		t.Fatalf("got %v", ids)
	}
}
