package catcher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	homerun "github.com/stuttgart-things/homerun-library/v4"
)

// WaitForReady retries probe with exponential backoff (1s, 2s, 4s, 8s, capped
// at 16s) until ctx is canceled or probe returns nil. Each attempt runs under
// a child context with timeout perAttemptTimeout.
//
// On success after retries, it logs at info level with the attempt count.
// On each failed attempt before the budget runs out, it logs at warn level.
// On budget exhaustion it returns the last probe error wrapped with the
// attempt count, leaving the os.Exit decision to the caller.
//
// Intended use: smooth over short readiness races at startup (Cilium identity
// propagation in a fresh namespace, a redis-stack installed alongside, etc.)
// without turning genuine misconfiguration into a silent hang.
//
// Same helper as homerun2-omni-pitcher's internal/pitcher/wait.go and
// homerun2-core-catcher's internal/catcher/wait.go; keep them in step until it
// moves into homerun-library (stuttgart-things/homerun-library#123).
func WaitForReady(ctx context.Context, probe func(context.Context) error, perAttemptTimeout time.Duration) error {
	const maxBackoff = 16 * time.Second
	backoff := time.Second

	var lastErr error
	for attempt := 1; ; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, perAttemptTimeout)
		err := probe(attemptCtx)
		cancel()
		if err == nil {
			if attempt > 1 {
				slog.Info("readiness probe succeeded after retries", "attempts", attempt)
			}
			return nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return fmt.Errorf("after %d attempts: %w", attempt, lastErr)
		}

		slog.Warn("readiness probe failed, retrying",
			"attempt", attempt,
			"error", err,
			"next_sleep", backoff.String(),
		)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return fmt.Errorf("after %d attempts: %w", attempt, lastErr)
		}

		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// redisPingTimeout bounds a single PING while waiting for Redis.
const redisPingTimeout = 5 * time.Second

// WaitForRedis blocks until Redis answers PING or timeout expires. The
// consumer's own preflight dials exactly once, so without this a Redis that is
// seconds away from ready makes the catcher exit (#59).
func WaitForRedis(rc homerun.RedisConfig, timeout time.Duration) error {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", rc.Addr, rc.Port),
		Password: rc.Password,
	})
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return WaitForReady(ctx, func(ctx context.Context) error {
		return client.Ping(ctx).Err()
	}, redisPingTimeout)
}
