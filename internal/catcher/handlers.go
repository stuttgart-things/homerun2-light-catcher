package catcher

import (
	"log/slog"
	"strconv"
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
			"timestamp", msg.Timestamp,
		)
	}
}

// LightHandler returns a MessageHandler that triggers WLED effects based on the profile.
func LightHandler(profilePath string, tracker *dashboard.EventTracker) MessageHandler {
	return func(msg models.CaughtMessage) {
		if !messageTimeValid(msg.Timestamp) {
			slog.Warn("message too old, skipping light trigger",
				"objectId", msg.ObjectID,
				"timestamp", msg.Timestamp,
			)
			return
		}

		wled.SendToWLED(profilePath, msg.Severity, msg.System, tracker)
	}
}

const maxTimeDiff int64 = 3

// messageTimeValid checks if a message timestamp is within maxTimeDiff seconds of now.
func messageTimeValid(timestamp string) bool {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		slog.Debug("invalid timestamp, allowing message", "timestamp", timestamp)
		return true
	}

	diff := time.Now().Unix() - ts
	return diff >= -maxTimeDiff && diff <= maxTimeDiff
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
