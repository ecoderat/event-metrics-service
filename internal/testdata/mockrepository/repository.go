package mockrepository

import (
	"context"

	"event-metrics-service/internal/model"
	"event-metrics-service/internal/repository"

	"github.com/stretchr/testify/mock"
)

type Repository struct {
	mock.Mock
}

// Interface compliance check
var _ repository.EventRepository = &Repository{}

func (m *Repository) Create(ctx context.Context, event model.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *Repository) CreateBatch(ctx context.Context, events []model.Event) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *Repository) FetchMetrics(ctx context.Context, filter model.MetricsFilter) (uint64, uint64, []model.MetricsGroup, error) {
	args := m.Called(ctx, filter)
	// Return type casting requires caution; ensure mocks are set up correctly in tests
	return args.Get(0).(uint64), args.Get(1).(uint64), args.Get(2).([]model.MetricsGroup), args.Error(3)
}
