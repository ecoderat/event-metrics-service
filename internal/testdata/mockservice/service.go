package mockservice

import (
	"context"

	"event-metrics-service/internal/model"

	"github.com/stretchr/testify/mock"
)

type Service struct {
	mock.Mock
}

func (m *Service) BuildEvent(req model.EventRequest) (model.Event, error) {
	args := m.Called(req)
	return args.Get(0).(model.Event), args.Error(1)
}

func (m *Service) ProcessEvent(ctx context.Context, event model.Event) (model.EventResult, error) {
	args := m.Called(ctx, event)
	return args.Get(0).(model.EventResult), args.Error(1)
}
