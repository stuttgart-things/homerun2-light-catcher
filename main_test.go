package main

import (
	"testing"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/catcher"
)

func TestCatcherInterface(t *testing.T) {
	t.Parallel()
	var _ catcher.Catcher = catcher.NewMockCatcher()
}

func TestMockCatcherRunAndShutdown(t *testing.T) {
	t.Parallel()
	mock := catcher.NewMockCatcher()

	done := make(chan struct{})
	go func() {
		mock.Run()
		close(done)
	}()

	// Wait for Run to actually start before shutting down
	<-mock.Started()
	mock.Shutdown()
	<-done
}
