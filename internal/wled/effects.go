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

// nameList caches one endpoint's list of effect or palette names.
type nameList struct {
	mu sync.Mutex
	// ids maps a lower-cased name to its ID; nil until a fetch succeeded.
	ids map[string]int
	// attemptedAt is the time of the last fetch, successful or not.
	attemptedAt time.Time
}

// The two lists WLED numbers by position.
const (
	effectsPath  = "/json/eff"
	palettesPath = "/json/pal"
)

var (
	nameListsMu sync.Mutex
	nameLists   = map[string]*nameList{}
)

func nameListFor(endpoint, path string) *nameList {
	nameListsMu.Lock()
	defer nameListsMu.Unlock()
	key := endpoint + path
	l, ok := nameLists[key]
	if !ok {
		l = &nameList{}
		nameLists[key] = l
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
func ResolveEffect(endpoint, fx string) (id int, source string, err error) {
	name := strings.TrimSpace(fx)
	if n, convErr := strconv.Atoi(name); convErr == nil {
		if n < 0 {
			return 0, "", fmt.Errorf("effect ID %d is negative", n)
		}
		return n, SourceNumeric, nil
	}

	id, known, err := lookupName(endpoint, effectsPath, name)
	if err == nil {
		return id, SourceDevice, nil
	}
	if known {
		return 0, "", fmt.Errorf("unknown effect %q: %w", name, err)
	}

	for n, id := range profile.FxMap {
		if strings.EqualFold(n, name) {
			return id, SourceFallback, nil
		}
	}
	return 0, "", fmt.Errorf("unknown effect %q: %s%s unreachable and not in the fallback table", name, endpoint, effectsPath)
}

// ResolvePalette returns the WLED palette ID for name on the given endpoint,
// looked up the same way as effects: a number is the ID itself, otherwise the
// device's /json/pal answers, case-insensitively. There is no fallback table --
// palette numbering has moved between WLED releases just like effects, and a
// wrong palette is worse than the colors the profile names locally.
func ResolvePalette(endpoint, palette string) (int, error) {
	name := strings.TrimSpace(palette)
	if id, err := strconv.Atoi(name); err == nil {
		if id < 0 {
			return 0, fmt.Errorf("palette ID %d is negative", id)
		}
		return id, nil
	}
	id, _, err := lookupName(endpoint, palettesPath, name)
	if err != nil {
		return 0, fmt.Errorf("unknown palette %q: %w", name, err)
	}
	return id, nil
}

// lookupName resolves name in the endpoint's list at path. known reports
// whether the device's list was available: when it was, a miss is final; when
// it was not, the caller may fall back.
func lookupName(endpoint, path, name string) (id int, known bool, err error) {
	l := nameListFor(endpoint, path)
	l.mu.Lock()
	defer l.mu.Unlock()

	key := strings.ToLower(name)
	// Every fetch is throttled, the first one included: a device that is down
	// must cost one request per refetchAfter, not one per message.
	due := l.attemptedAt.IsZero() || time.Since(l.attemptedAt) >= refetchAfter
	if l.ids == nil && due {
		l.fetch(endpoint, path)
		due = false
	}
	if id, ok := l.ids[key]; ok {
		return id, true, nil
	}
	// Unknown to the cached list: the device may have been updated since.
	if l.ids != nil && due {
		l.fetch(endpoint, path)
		if id, ok := l.ids[key]; ok {
			return id, true, nil
		}
	}
	if l.ids != nil {
		return 0, true, fmt.Errorf("not in %s%s", endpoint, path)
	}
	return 0, false, fmt.Errorf("%s%s unreachable", endpoint, path)
}

// fetch loads the endpoint's list at path. On failure the previous list, if
// any, is kept. Callers hold l.mu.
func (l *nameList) fetch(endpoint, path string) {
	l.attemptedAt = time.Now()
	names, err := fetchNames(endpoint, path)
	if err != nil {
		return
	}
	l.ids = effectIDs(names)
}

// fetchNames reads GET {endpoint}{path}: a JSON array of names whose index is
// the ID (/json/eff for effects, /json/pal for palettes).
func fetchNames(endpoint, path string) ([]string, error) {
	resp, err := effectLookupClient.Get(endpoint + path)
	if err != nil {
		return nil, fmt.Errorf("GET %s%s: %w", endpoint, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s%s: %s", endpoint, path, resp.Status)
	}
	var names []string
	if err := json.NewDecoder(resp.Body).Decode(&names); err != nil {
		return nil, fmt.Errorf("decode %s%s: %w", endpoint, path, err)
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
