package catcher

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/dashboard"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/models"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/wled"
)

// LogHandler returns a MessageHandler that logs messages with severity-aware levels.
func LogHandler() MessageHandler {
	return func(msg models.CaughtMessage) {
		level := severityToLevel(msg.Severity)

		slog.Log(nil, level, "message caught",
			"objectId", msg.ObjectID,
			"streamId", msg.StreamID,
			"title", msg.Title,
			"severity", msg.Severity,
			"system", msg.System,
			"tags", msg.Tags,
			"timestamp", msg.Timestamp,
		)
	}
}

// LightHandler returns a MessageHandler that triggers WLED effects based on the profile.
//
// Messages pitched to the stream more than maxAge ago are skipped: a light is a
// signal about now, and a late delivery (a backlog after a long outage) would
// only replay stale effects. Age is measured from the stream entry, not from the
// message's timestamp field, which producers set to when the event happened
// (git-pitcher, for example, uses the GitHub event's creation time, minutes
// before it is pitched). maxAge 0 disables the check.
func LightHandler(profilePath string, maxAge time.Duration, tracker *dashboard.EventTracker) MessageHandler {
	return func(msg models.CaughtMessage) {
		if maxAge > 0 {
			pitched, err := streamEntryTime(msg.StreamID)
			if err != nil {
				slog.Warn("cannot tell when message was pitched, allowing it",
					"objectId", msg.ObjectID,
					"streamId", msg.StreamID,
					"error", err,
				)
			} else if age := time.Since(pitched); age > maxAge {
				slog.Warn("message pitched too long ago, skipping light trigger",
					"objectId", msg.ObjectID,
					"streamId", msg.StreamID,
					"age", age.Round(time.Millisecond).String(),
					"max_age", maxAge.String(),
				)
				return
			}
		}

		wled.SendToWLED(profilePath, msg.Severity, msg.System, msg.Tags, tracker)
	}
}

// streamEntryTime returns when Redis added a stream entry, taken from the
// millisecond part of its ID ("1789100802521-0").
func streamEntryTime(id string) (time.Time, error) {
	ms, _, _ := strings.Cut(id, "-")
	n, err := strconv.ParseInt(ms, 10, 64)
	if err != nil || n < 0 {
		return time.Time{}, fmt.Errorf("not a stream entry ID: %q", id)
	}
	return time.UnixMilli(n), nil
}

func severityToLevel(severity string) slog.Level {
	switch severity {
	case "error", "ERROR":
		return slog.LevelError
	case "warning", "WARNING":
		return slog.LevelWarn
	case "debug", "DEBUG":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}
