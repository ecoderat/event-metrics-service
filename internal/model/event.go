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

// EventResult describes the outcome of an insert.
type EventResult struct {
	Status string  `json:"status"`
	ID     *uint64 `json:"id,omitempty"`
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
	TotalCount      uint64 `json:"total_count"`
	UniqueUserCount uint64 `json:"unique_user_count"`
}

// MetricsResponse is returned to clients for metrics queries.
type MetricsResponse struct {
	Meta MetricsMeta `json:"meta"`
	Data MetricsData `json:"data"`
}

// MetricsMeta contains metadata about the metrics query.
type MetricsMeta struct {
	EventName string                 `json:"event_name"`
	Period    MetricsPeriod          `json:"period"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	GroupBy   string                 `json:"group_by,omitempty"`
}

// MetricsPeriod captures the time window.
type MetricsPeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// MetricsData holds aggregated values.
type MetricsData struct {
	TotalEventCount  uint64         `json:"total_event_count"`
	UniqueEventCount uint64         `json:"unique_event_count"`
	Groups           []MetricsGroup `json:"groups,omitempty"`
}
