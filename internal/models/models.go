package models

import (
	"time"

	homerun "github.com/stuttgart-things/homerun-library/v2"
)

// CaughtMessage wraps a homerun.Message with stream metadata.
type CaughtMessage struct {
	homerun.Message
	ObjectID string    `json:"objectId"`
	StreamID string    `json:"streamId"`
	CaughtAt time.Time `json:"caughtAt"`
}
