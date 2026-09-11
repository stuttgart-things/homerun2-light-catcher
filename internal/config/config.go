package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	homerun "github.com/stuttgart-things/homerun-library/v4"
)

func LoadRedisConfig() homerun.RedisConfig {
	return homerun.RedisConfig{
		Addr:     homerun.GetEnv("REDIS_ADDR", "localhost"),
		Port:     homerun.GetEnv("REDIS_PORT", "6379"),
		Password: homerun.GetEnv("REDIS_PASSWORD", ""),
		Stream:   homerun.GetEnv("REDIS_STREAM", "messages"),
	}
}

// LoadStreams returns the list of Redis streams the catcher should subscribe to.
//
// Resolution:
//  1. REDIS_STREAMS (comma-separated) — preferred, enables multi-stream subscription
//  2. REDIS_STREAM — legacy single-stream env var, wrapped in a one-element list
//  3. ["messages"] — hardcoded default
//
// Legacy single-stream deployments keep working unchanged.
func LoadStreams() []string {
	return ParseStreams(os.Getenv("REDIS_STREAMS"), os.Getenv("REDIS_STREAM"))
}

// ParseStreams is the pure resolution helper behind LoadStreams. Exposed for tests.
func ParseStreams(streamsEnv, streamFallback string) []string {
	if streamsEnv != "" {
		parts := strings.Split(streamsEnv, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if streamFallback != "" {
		return []string{streamFallback}
	}
	return []string{"messages"}
}

// DefaultRedisStartupTimeout is how long startup waits for Redis to answer.
// A freshly installed redis-stack took ~70s on labda-dev-a (2026-09-10).
const DefaultRedisStartupTimeout = 120 * time.Second

// LoadRedisStartupTimeout reads REDIS_STARTUP_TIMEOUT (a Go duration, e.g.
// "90s" or "2m"). Unset means DefaultRedisStartupTimeout.
func LoadRedisStartupTimeout() (time.Duration, error) {
	return ParseRedisStartupTimeout(os.Getenv("REDIS_STARTUP_TIMEOUT"))
}

// ParseRedisStartupTimeout returns an error for an unparsable or
// non-positive value rather than falling back: a typo here should fail
// startup loudly, not quietly restore a budget nobody chose.
func ParseRedisStartupTimeout(v string) (time.Duration, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return DefaultRedisStartupTimeout, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("REDIS_STARTUP_TIMEOUT %q: %w", v, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("REDIS_STARTUP_TIMEOUT %q: must be positive", v)
	}
	return d, nil
}

// DefaultMaxMessageAge is how long after being pitched a message still
// triggers a light: long enough to survive a pod restart, short enough that a
// backlog from a longer outage is not replayed.
const DefaultMaxMessageAge = 60 * time.Second

// LoadMaxMessageAge reads MAX_MESSAGE_AGE.
func LoadMaxMessageAge() (time.Duration, error) {
	return ParseMaxMessageAge(os.Getenv("MAX_MESSAGE_AGE"))
}

// ParseMaxMessageAge parses a Go duration ("60s", "2m"). Empty means
// DefaultMaxMessageAge, "0" disables the age check. Exposed for tests.
func ParseMaxMessageAge(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultMaxMessageAge, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid MAX_MESSAGE_AGE %q: use a duration such as 60s or 2m, or 0 to disable", value)
	}
	if d < 0 {
		return 0, fmt.Errorf("invalid MAX_MESSAGE_AGE %q: must not be negative", value)
	}
	return d, nil
}

// SetupLogging configures slog as the default logger based on LOG_FORMAT and LOG_LEVEL env vars.
func SetupLogging() {
	format := strings.ToLower(homerun.GetEnv("LOG_FORMAT", "json"))
	levelStr := strings.ToLower(homerun.GetEnv("LOG_LEVEL", "info"))

	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))

	// homerun-library is silent by default as of v3.2.0. Routing it through the
	// same logger means its records arrive in this service's format and at this
	// service's level, instead of the pterm-decorated stdout writes it used to
	// interleave into the log stream.
	homerun.SetLogger(slog.Default())
}
