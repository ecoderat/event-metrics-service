package service

import (
	"context"
	"errors"
	"time"

	"event-metrics-service/internal/model"
	"event-metrics-service/internal/repository"
)

// ValidationError represents user input issues.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// eventService wires business logic for events and metrics.
type eventService struct {
	repo            repository.EventRepository
	worker          BatchEventWorker
	now             func() time.Time
	futureTolerance time.Duration
}

type EventService interface {
	BuildEvent(req model.EventRequest) (model.Event, error)
	ProcessEvent(ctx context.Context, event model.Event) (model.EventResult, error)
}

// NewEventService constructs an eventService.
func NewEventService(repo repository.EventRepository, worker BatchEventWorker, futureTolerance time.Duration) EventService {
	return &eventService{
		repo:            repo,
		worker:          worker,
		now:             time.Now,
		futureTolerance: futureTolerance,
	}
}

// BuildEvent validates and constructs an Event from an incoming request.
func (s *eventService) BuildEvent(req model.EventRequest) (model.Event, error) {
	if req.EventName == "" {
		return model.Event{}, &ValidationError{Message: "event_name is required"}
	}

	if req.Channel == "" {
		return model.Event{}, &ValidationError{Message: "channel is required"}
	}

	if req.UserID == "" {
		return model.Event{}, &ValidationError{Message: "user_id is required"}
	}

	if req.Timestamp == 0 {
		return model.Event{}, &ValidationError{Message: "timestamp is required"}
	}

	ts := time.Unix(req.Timestamp, 0).UTC()
	if s.futureTolerance > 0 {
		if err := ValidateTimestamp(ts, s.now(), s.futureTolerance); err != nil {
			return model.Event{}, &ValidationError{Message: err.Error()}
		}
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	campaignID := ""
	if req.CampaignID != nil {
		campaignID = *req.CampaignID
	}

	event := model.Event{
		EventName:  req.EventName,
		Channel:    req.Channel,
		CampaignID: campaignID,
		UserID:     req.UserID,
		Timestamp:  ts,
		Tags:       tags,
		Metadata:   req.Metadata,
	}

	return event, nil
}

// ProcessEvent persists a single event.
func (s *eventService) ProcessEvent(ctx context.Context, event model.Event) (model.EventResult, error) {
	s.worker.Enqueue(event)
	return model.EventResult{Status: "created"}, nil
}

// ValidateTimestamp ensures timestamps are not too far in the future.
func ValidateTimestamp(ts time.Time, now time.Time, tolerance time.Duration) error {
	if tolerance <= 0 {
		return nil
	}
	if ts.After(now.Add(tolerance)) {
		return errors.New("timestamp cannot be in the future")
	}
	return nil
}
