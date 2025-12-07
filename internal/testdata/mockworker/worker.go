package mockworker

import (
	"event-metrics-service/internal/model"

	"github.com/stretchr/testify/mock"
)

type Worker struct {
	mock.Mock
}

func (m *Worker) Enqueue(event model.Event) {
	m.Called(event)
}

func (m *Worker) Shutdown() {
	m.Called()
}
