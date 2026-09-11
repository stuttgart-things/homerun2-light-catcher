package config

import (
	"reflect"
	"testing"
	"time"
)

func TestParseStreams(t *testing.T) {
	cases := []struct {
		name       string
		streamsEnv string
		streamFall string
		want       []string
	}{
		{"multi from REDIS_STREAMS", "homerun,releases", "ignored", []string{"homerun", "releases"}},
		{"whitespace trimmed", " homerun , releases ", "", []string{"homerun", "releases"}},
		{"empty entries dropped", "homerun,,releases,", "", []string{"homerun", "releases"}},
		{"legacy REDIS_STREAM single", "", "homerun", []string{"homerun"}},
		{"empty streams env falls back to legacy", "", "messages", []string{"messages"}},
		{"both unset returns hardcoded default", "", "", []string{"messages"}},
		{"streams with only whitespace falls back to legacy", " , , ", "legacy", []string{"legacy"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseStreams(tc.streamsEnv, tc.streamFall)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseStreams(%q, %q) = %v, want %v", tc.streamsEnv, tc.streamFall, got, tc.want)
			}
		})
	}
}

func TestParseMaxMessageAge(t *testing.T) {
	cases := []struct {
		value   string
		want    time.Duration
		wantErr bool
	}{
		{"", DefaultMaxMessageAge, false},
		{"  ", DefaultMaxMessageAge, false},
		{"60s", time.Minute, false},
		{"2m", 2 * time.Minute, false},
		{"1500ms", 1500 * time.Millisecond, false},
		{"0", 0, false},
		{"60", 0, true},
		{"soon", 0, true},
		{"-5s", 0, true},
	}
	for _, tc := range cases {
		got, err := ParseMaxMessageAge(tc.value)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseMaxMessageAge(%q) error = %v, wantErr %v", tc.value, err, tc.wantErr)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseMaxMessageAge(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}
