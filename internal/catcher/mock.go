package catcher

import "sync"

// MockCatcher is a no-op catcher for testing.
type MockCatcher struct {
	done    chan struct{}
	started chan struct{}
	once    sync.Once
}

func NewMockCatcher() *MockCatcher {
	return &MockCatcher{
		done:    make(chan struct{}),
		started: make(chan struct{}),
	}
}

func (m *MockCatcher) Run() {
	close(m.started)
	<-m.done
}

func (m *MockCatcher) Shutdown() {
	m.once.Do(func() { close(m.done) })
}

// Started returns a channel that is closed when Run() begins.
func (m *MockCatcher) Started() <-chan struct{} {
	return m.started
}

func (m *MockCatcher) Errors() <-chan error {
	return nil
}
