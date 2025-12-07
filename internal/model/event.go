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

// BulkEventsRequest holds a list of events for bulk ingestion.
type BulkEventsRequest struct {
	Events []EventRequest `json:"events"`
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

// EventResult describes the outcome of an insert.
type EventResult struct {
	Status string `json:"status"`
	ID     *int64 `json:"id,omitempty"`
}

// BulkInsertResult summarizes a bulk ingestion attempt.
type BulkInsertResult struct {
	Inserted   int         `json:"inserted"`
	Duplicates int         `json:"duplicates"`
	Failed     int         `json:"failed"`
	Errors     []BulkError `json:"errors,omitempty"`
}

// BulkError captures per-item validation failures.
type BulkError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

// MetricsFilter represents metrics query filters.
type MetricsFilter struct {
	EventName string
	From      time.Time
	To        time.Time
	Channel   *string
	GroupBy   string
}

// MetricsGroup is a grouped metrics result.
type MetricsGroup struct {
	Key             string `json:"key"`
	TotalCount      int64  `json:"total_count"`
	UniqueUserCount int64  `json:"unique_user_count"`
}

// MetricsResponse is returned to clients for metrics queries.
type MetricsResponse struct {
	EventName       string                 `json:"event_name"`
	From            int64                  `json:"from"`
	To              int64                  `json:"to"`
	Filter          map[string]interface{} `json:"filter,omitempty"`
	TotalCount      int64                  `json:"total_count"`
	UniqueUserCount int64                  `json:"unique_user_count"`
	GroupBy         string                 `json:"group_by,omitempty"`
	Groups          []MetricsGroup         `json:"groups,omitempty"`
}
