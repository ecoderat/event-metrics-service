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

func (m *Service) ProcessEvent(ctx context.Context, event model.Event) {
	m.Called(ctx, event)
}

func (m *Service) GetMetrics(ctx context.Context, filter model.MetricsFilter) (model.MetricsResponse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(model.MetricsResponse), args.Error(1)
}
