package wled

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/profile"
)

// Effect IDs are looked up on the device, not taken from a table.
//
// WLED numbers its effects by position in /json/eff, and the numbering is not
// stable across releases: against a real WLED 16.0.1 most of profile.FxMap was
// wrong -- "Twinkle" (5) showed Random Colors, "Rainbow" (7) Dynamic -- and a
// name outside the table, such as "Strobe", lit nothing at all (#77). The
// device's own list is the only numbering that is right for that device.

// effectLookupClient is short on purpose: the lookup sits in front of every
// effect, and an unreachable device must not hold a message for the full
// httpClient timeout.
var effectLookupClient = &http.Client{Timeout: 3 * time.Second}

// refetchAfter throttles reloads of the effect list. An unknown name reloads
// it (the device may have been updated), and so does a device that could not
// be reached -- but at most once per interval, so a typo in a profile or a
// device that is down costs one request per minute, not one per message.
var refetchAfter = time.Minute

// effectList caches one endpoint's effect list.
type effectList struct {
	mu sync.Mutex
	// ids maps a lower-cased effect name to its ID; nil until a fetch succeeded.
	ids map[string]int
	// attemptedAt is the time of the last fetch, successful or not.
	attemptedAt time.Time
}

var (
	effectListsMu sync.Mutex
	effectLists   = map[string]*effectList{}
)

func effectListFor(endpoint string) *effectList {
	effectListsMu.Lock()
	defer effectListsMu.Unlock()
	l, ok := effectLists[endpoint]
	if !ok {
		l = &effectList{}
		effectLists[endpoint] = l
	}
	return l
}

// Effect sources, reported by ResolveEffect.
const (
	SourceNumeric  = "numeric"
	SourceDevice   = "device"
	SourceFallback = "fallback"
)

// ResolveEffect returns the WLED effect ID for fx on the given endpoint, and
// where the ID came from.
//
//   - A numeric fx is taken as the ID itself -- the escape hatch for an effect
//     whose name is ambiguous or not worth a lookup.
//   - Otherwise the name is looked up, case-insensitively, in the device's
//     /json/eff. That list is authoritative: a name the device does not know
//     is an error, even if profile.FxMap has it.
//   - Only when the device's list could not be fetched does profile.FxMap
//     answer, so a device that is briefly unreachable still gets a best guess.
func ResolveEffect(endpoint, fx string) (int, string, error) {
	name := strings.TrimSpace(fx)
	if id, err := strconv.Atoi(name); err == nil {
		if id < 0 {
			return 0, "", fmt.Errorf("effect ID %d is negative", id)
		}
		return id, SourceNumeric, nil
	}

	l := effectListFor(endpoint)
	l.mu.Lock()
	defer l.mu.Unlock()

	key := strings.ToLower(name)
	// Every fetch is throttled, the first one included: a device that is down
	// must cost one request per refetchAfter, not one per message.
	due := l.attemptedAt.IsZero() || time.Since(l.attemptedAt) >= refetchAfter
	if l.ids == nil && due {
		l.fetch(endpoint)
		due = false
	}
	if id, ok := l.ids[key]; ok {
		return id, SourceDevice, nil
	}
	// Unknown to the cached list: the device may have been updated since.
	if l.ids != nil && due {
		l.fetch(endpoint)
		if id, ok := l.ids[key]; ok {
			return id, SourceDevice, nil
		}
	}
	if l.ids != nil {
		return 0, "", fmt.Errorf("unknown effect %q: not in %s/json/eff", name, endpoint)
	}

	for n, id := range profile.FxMap {
		if strings.EqualFold(n, name) {
			return id, SourceFallback, nil
		}
	}
	return 0, "", fmt.Errorf("unknown effect %q: %s/json/eff unreachable and not in the fallback table", name, endpoint)
}

// fetch loads the endpoint's effect list. On failure the previous list, if
// any, is kept. Callers hold l.mu.
func (l *effectList) fetch(endpoint string) {
	l.attemptedAt = time.Now()
	names, err := fetchEffectNames(endpoint)
	if err != nil {
		return
	}
	l.ids = effectIDs(names)
}

// fetchEffectNames reads GET {endpoint}/json/eff: a JSON array of effect
// names whose index is the effect ID.
func fetchEffectNames(endpoint string) ([]string, error) {
	resp, err := effectLookupClient.Get(endpoint + "/json/eff")
	if err != nil {
		return nil, fmt.Errorf("GET %s/json/eff: %w", endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s/json/eff: %s", endpoint, resp.Status)
	}
	var names []string
	if err := json.NewDecoder(resp.Body).Decode(&names); err != nil {
		return nil, fmt.Errorf("decode %s/json/eff: %w", endpoint, err)
	}
	return names, nil
}

// effectIDs maps lower-cased names to their index. WLED marks removed or
// unavailable slots "RSVD" (and some builds "-"); those are not names. On a
// duplicate name the lowest ID wins.
func effectIDs(names []string) map[string]int {
	ids := make(map[string]int, len(names))
	for i, n := range names {
		key := strings.ToLower(strings.TrimSpace(n))
		if key == "" || key == "rsvd" || key == "-" {
			continue
		}
		if _, dup := ids[key]; !dup {
			ids[key] = i
		}
	}
	return ids
}
