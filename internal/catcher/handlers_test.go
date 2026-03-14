package catcher

import (
	"fmt"
	"testing"
	"time"
)

func TestMessageTimeValid_Current(t *testing.T) {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	if !messageTimeValid(ts) {
		t.Error("current timestamp should be valid")
	}
}

func TestMessageTimeValid_TooOld(t *testing.T) {
	ts := fmt.Sprintf("%d", time.Now().Unix()-10)
	if messageTimeValid(ts) {
		t.Error("10s old timestamp should be invalid with 3s window")
	}
}

func TestMessageTimeValid_InvalidFormat(t *testing.T) {
	if !messageTimeValid("not-a-number") {
		t.Error("invalid timestamp should be allowed (returns true)")
	}
}

func TestSeverityToLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ERROR", "ERROR"},
		{"error", "ERROR"},
		{"WARNING", "WARN"},
		{"warning", "WARN"},
		{"INFO", "INFO"},
		{"SUCCESS", "INFO"},
		{"DEBUG", "DEBUG"},
	}

	for _, tt := range tests {
		level := severityToLevel(tt.input)
		if level.String() != tt.expected {
			t.Errorf("severityToLevel(%q) = %s, want %s", tt.input, level.String(), tt.expected)
		}
	}
}
