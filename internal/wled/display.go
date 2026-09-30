package wled

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// A duration of "auto" keeps the light on while the led-catcher's panel shows
// something, polling its GET /display (#77). The matrix scrolls a text for
// (64 + text width) x 30 ms -- a width only the led-catcher knows, from its own
// profile and font -- and drops messages that arrive while it is busy, so a
// fixed duration drifted apart from it in every burst.

// ledCatcherURLEnv is the led-catcher base URL for effects that set no
// display of their own.
const ledCatcherURLEnv = "LED_CATCHER_URL"

// Tuning for "auto"; tests shorten them.
var (
	// displayPollInterval is how often the panel is asked what it shows.
	displayPollInterval = 250 * time.Millisecond
	// displayStartGrace is how long the panel may take to start showing the
	// message: the led-catcher reads the same stream, not this effect.
	displayStartGrace = 3 * time.Second
	// displayUnreachable ends the effect when the panel cannot be asked at all.
	displayUnreachable = 5 * time.Second
	// autoMaxDuration caps the light for a held display or a panel that never
	// finishes.
	autoMaxDuration = 2 * time.Minute
)

// displayState is the part of the led-catcher's GET /display this needs.
type displayState struct {
	Showing bool `json:"showing"`
}

// displayURL returns the led-catcher base URL for an effect.
func displayURL(display string) string {
	if display == "" {
		display = os.Getenv(ledCatcherURLEnv)
	}
	return strings.TrimRight(display, "/")
}

func fetchDisplay(base string) (displayState, error) {
	var st displayState
	resp, err := effectLookupClient.Get(base + "/display")
	if err != nil {
		return st, fmt.Errorf("GET %s/display: %w", base, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return st, fmt.Errorf("GET %s/display: %s", base, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return st, fmt.Errorf("decode %s/display: %w", base, err)
	}
	return st, nil
}

// followDisplay blocks until the effect with the given generation should end:
// the panel has shown something and stopped, it never started within
// displayStartGrace, it could not be reached for displayUnreachable, or
// autoMaxDuration is over. It returns early once a newer effect was sent to
// the device -- that effect follows the panel from then on.
//
// If the panel is still busy with an earlier message when this one arrives,
// the led-catcher drops this one and the light stays with what is shown: they
// end together, which is the point.
func followDisplay(base string, dev *device, generation uint64) (reason string) {
	start := time.Now()
	seen := false
	var lastErr error
	for {
		dev.mu.Lock()
		superseded := dev.generation != generation
		dev.mu.Unlock()
		if superseded {
			return "superseded"
		}

		elapsed := time.Since(start)
		if elapsed >= autoMaxDuration {
			return "max duration"
		}

		st, err := fetchDisplay(base)
		switch {
		case err != nil:
			lastErr = err
			if !seen && elapsed >= displayUnreachable {
				slog.Warn("led-catcher panel unreachable, ending the effect", "display", base, "error", lastErr)
				return "unreachable"
			}
		case st.Showing:
			seen = true
		case seen:
			return "panel finished"
		case elapsed >= displayStartGrace:
			return "panel never started"
		}
		time.Sleep(displayPollInterval)
	}
}
