package catcher

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	homerun "github.com/stuttgart-things/homerun-library/v4"
)

func TestWaitForReady_FirstTry(t *testing.T) {
	var calls atomic.Int32
	probe := func(ctx context.Context) error {
		calls.Add(1)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := WaitForReady(ctx, probe, 100*time.Millisecond); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("probe calls = %d, want 1 (happy path must not retry)", got)
	}
}

func TestWaitForReady_SucceedsAfterRetries(t *testing.T) {
	var calls atomic.Int32
	probe := func(ctx context.Context) error {
		n := calls.Add(1)
		if n < 3 {
			return errors.New("not ready yet")
		}
		return nil
	}

	// Use a short stand-in: WaitForReady's first backoff is 1s. Budget of 5s
	// is enough for two retry sleeps (1s + 2s) plus a hair of slack.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	if err := WaitForReady(ctx, probe, 100*time.Millisecond); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("probe calls = %d, want 3", got)
	}
	// First sleep 1s + second sleep 2s = 3s of backoff before the third probe.
	if elapsed := time.Since(start); elapsed < 2900*time.Millisecond {
		t.Errorf("elapsed = %v, want at least 2.9s (skipped backoff?)", elapsed)
	}
}

func TestWaitForReady_BudgetExhausted(t *testing.T) {
	var calls atomic.Int32
	probe := func(ctx context.Context) error {
		calls.Add(1)
		return errors.New("redis unreachable")
	}

	// 50ms budget, 10ms per attempt — guarantees we hit the deadline quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := WaitForReady(ctx, probe, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when budget exhausted, got nil")
	}
	if !strings.Contains(err.Error(), "redis unreachable") {
		t.Errorf("error %q does not wrap last probe error", err.Error())
	}
	if !strings.Contains(err.Error(), "after") {
		t.Errorf("error %q missing attempt-count prefix", err.Error())
	}
	if calls.Load() < 1 {
		t.Error("probe never called")
	}
}

func TestWaitForReady_StopsImmediatelyOnContextCancel(t *testing.T) {
	probe := func(ctx context.Context) error {
		return errors.New("boom")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	start := time.Now()
	err := WaitForReady(ctx, probe, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when ctx already canceled")
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Errorf("elapsed = %v, want <200ms (did the helper sleep through cancel?)", elapsed)
	}
}

func TestWaitForRedis_GivesUpAtTimeout(t *testing.T) {
	// Nothing listens on port 1; the dial is refused at once, so this needs
	// no Redis and only checks that the budget bounds the wait.
	rc := homerun.RedisConfig{Addr: "127.0.0.1", Port: "1"}
	start := time.Now()
	err := WaitForRedis(rc, 1500*time.Millisecond)
	if err == nil {
		t.Fatal("expected an error with no Redis listening")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("WaitForRedis took %v, want it bounded by the 1.5s budget", elapsed)
	}
}
