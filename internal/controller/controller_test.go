package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"event-metrics-service/internal/model"

	mockservice "event-metrics-service/internal/testdata/mockservice"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ControllerTestSuite struct {
	suite.Suite
	app     *fiber.App
	service *mockservice.Service
}

func TestControllerSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}

func (s *ControllerTestSuite) SetupTest() {
	s.service = &mockservice.Service{}
	ctrl := NewEventController(s.service)
	s.app = fiber.New()
	s.app.Post("/events", ctrl.CreateEvent)
	s.app.Get("/metrics", ctrl.GetMetrics)
}

func (s *ControllerTestSuite) TestCreateEvent_Success() {
	now := time.Unix(100, 0).UTC()
	reqBody := model.EventRequest{
		EventName: "signup",
		Channel:   "web",
		UserID:    "u1",
		Timestamp: now.Unix(),
	}
	ev := model.Event{
		EventName: "signup",
		Channel:   "web",
		UserID:    "u1",
		Timestamp: now,
	}
	s.service.On("BuildEvent", reqBody).Return(ev, nil)
	s.service.On("ProcessEvent", mock.Anything, ev).Return(nil)

	resp := s.performRequest(reqBody)

	require.Equal(s.T(), http.StatusAccepted, resp.StatusCode)
}

func (s *ControllerTestSuite) TestCreateEvent_InvalidJSON() {
	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString("{"))
	resp, _ := s.app.Test(req, -1)
	require.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
}

func (s *ControllerTestSuite) TestCreateEvent_BuildError() {
	now := time.Unix(100, 0).UTC()
	reqBody := model.EventRequest{
		EventName: "",
		Channel:   "web",
		UserID:    "u1",
		Timestamp: now.Unix(),
	}
	s.service.On("BuildEvent", reqBody).Return(model.Event{}, fiber.ErrBadRequest)

	resp := s.performRequest(reqBody)

	require.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
}

func (s *ControllerTestSuite) TestCreateEvent_ProcessError() {
	now := time.Unix(100, 0).UTC()
	reqBody := model.EventRequest{
		EventName: "signup",
		Channel:   "web",
		UserID:    "u1",
		Timestamp: now.Unix(),
	}
	ev := model.Event{EventName: "signup", Channel: "web", UserID: "u1", Timestamp: now}
	s.service.On("BuildEvent", reqBody).Return(ev, nil)
	s.service.On("ProcessEvent", mock.Anything, ev).Return(context.DeadlineExceeded)

	resp := s.performRequest(reqBody)

	require.Equal(s.T(), http.StatusInternalServerError, resp.StatusCode)
}

func (s *ControllerTestSuite) TestGetMetrics_Success() {
	filterMatcher := mock.MatchedBy(func(f model.MetricsFilter) bool {
		return f.EventName == "signup" && f.GroupBy == "channel"
	})
	expected := model.MetricsResponse{
		Meta: model.MetricsMeta{
			EventName: "signup",
			GroupBy:   "channel",
			Period:    model.MetricsPeriod{Start: time.Unix(0, 0).UTC().Format(time.RFC3339), End: time.Unix(0, 0).UTC().Format(time.RFC3339)},
		},
		Data: model.MetricsData{
			TotalEventCount:  10,
			UniqueEventCount: 3,
			Groups: []model.MetricsGroup{
				{Key: "web", TotalCount: 8, UniqueUserCount: 2},
			},
		},
	}
	s.service.On("GetMetrics", mock.Anything, filterMatcher).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/metrics?event_name=signup", nil)
	resp, err := s.app.Test(req, -1)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

func (s *ControllerTestSuite) TestGetMetrics_MissingEventName() {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp, err := s.app.Test(req, -1)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
}

func (s *ControllerTestSuite) TestGetMetrics_InvalidFrom() {
	req := httptest.NewRequest(http.MethodGet, "/metrics?event_name=signup&from=not-a-time", nil)
	resp, err := s.app.Test(req, -1)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
}

func (s *ControllerTestSuite) performRequest(body any) *http.Response {
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.app.Test(req, -1)
	require.NoError(s.T(), err)
	return resp
}
