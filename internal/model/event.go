package model

import (
	"time"
)

// EventRequest represents incoming event payload.
type EventRequest struct {
	EventName  string                 `json:"event_name"`
	Channel    string                 `json:"channel"`
	CampaignID *string                `json:"campaign_id"`
	UserID     string                 `json:"user_id"`
	Timestamp  int64                  `json:"timestamp"`
	Tags       []string               `json:"tags"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Event is the domain model persisted in the database.
type Event struct {
	ID         int64
	EventName  string
	Channel    string
	CampaignID string
	UserID     string
	Timestamp  time.Time
	Tags       []string
	Metadata   map[string]interface{}
}
