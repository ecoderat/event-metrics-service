package service

import (
	"context"
	"testing"
	"time"

	"event-metrics-service/internal/model"

	// Adjust these paths based on your actual project structure
	mockrepository "event-metrics-service/internal/testdata/mockrepository"
	mockworker "event-metrics-service/internal/testdata/mockworker"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type EventServiceTestSuite struct {
	suite.Suite

	repo   *mockrepository.Repository
	worker *mockworker.Worker

	// We hold a pointer to the concrete struct (not just the interface)
	// to access private fields like 'now' and 'futureTolerance' during testing.
	service *eventService
}

func TestEventServiceSuite(t *testing.T) {
	suite.Run(t, new(EventServiceTestSuite))
}

func (s *EventServiceTestSuite) SetupTest() {
	s.repo = &mockrepository.Repository{}
	s.worker = &mockworker.Worker{}

	// Initialize the service and cast it to the concrete struct
	svc := NewEventService(s.repo, s.worker, 0)
	s.service = svc.(*eventService)

	// Freeze time to a deterministic value for all tests
	s.service.now = func() time.Time { return time.Unix(1000, 0).UTC() }
}

// TestBuildEvent_ValidationErrors uses table-driven tests to verify all input constraints.
func (s *EventServiceTestSuite) TestBuildEvent_ValidationErrors() {
	tests := []struct {
		name      string
		req       model.EventRequest
		errMsg    string
		tolerance time.Duration
	}{
		{
			name:   "Missing EventName",
			req:    model.EventRequest{Channel: "web", UserID: "u1", Timestamp: 1000},
			errMsg: "event_name is required",
		},
		{
			name:   "Missing Channel",
			req:    model.EventRequest{EventName: "login", UserID: "u1", Timestamp: 1000},
			errMsg: "channel is required",
		},
		{
			name:   "Missing UserID",
			req:    model.EventRequest{EventName: "login", Channel: "web", Timestamp: 1000},
			errMsg: "user_id is required",
		},
		{
			name:   "Missing Timestamp",
			req:    model.EventRequest{EventName: "login", Channel: "web", UserID: "u1"}, // Ts defaults to 0
			errMsg: "timestamp is required",
		},
		{
			name: "Future Timestamp Error",
			req: model.EventRequest{
				EventName: "login", Channel: "web", UserID: "u1",
				Timestamp: 1005, // 5 seconds in the future relative to frozen time (1000)
			},
			errMsg:    "timestamp cannot be in the future",
			tolerance: 2 * time.Second, // Only allow 2 seconds of tolerance
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Set tolerance specifically for this test case
			s.service.futureTolerance = tt.tolerance

			_, err := s.service.BuildEvent(tt.req)

			s.Error(err)
			// Verify that the error is of the specific custom type
			s.IsType(&ValidationError{}, err)
			// Verify the error message matches exactly
			s.EqualError(err, tt.errMsg)
		})
	}
}

// TestBuildEvent_SuccessLogic verifies that the Event struct is constructed correctly,
// handling pointers and nil slices appropriately.
func (s *EventServiceTestSuite) TestBuildEvent_SuccessLogic() {
	campaignID := "summer_sale"
	req := model.EventRequest{
		EventName:  "purchase",
		Channel:    "mobile",
		UserID:     "user_123",
		Timestamp:  1000, // Matches the frozen 'now' time
		CampaignID: &campaignID,
		Tags:       nil, // Passing nil to ensure it converts to empty slice
		Metadata:   map[string]any{"price": 100},
	}

	event, err := s.service.BuildEvent(req)

	s.NoError(err)
	s.Equal("purchase", event.EventName)
	s.Equal("summer_sale", event.CampaignID, "Pointer value should be dereferenced correctly")
	s.NotNil(event.Tags, "Tags should not be nil")
	s.Empty(event.Tags, "Tags should be an empty slice, not nil")
	s.Equal(time.Unix(1000, 0).UTC(), event.Timestamp)
}

// TestBuildEvent_FutureToleranceDisabled verifies that future dates are accepted
// when tolerance is set to 0.
func (s *EventServiceTestSuite) TestBuildEvent_FutureToleranceDisabled() {
	s.service.futureTolerance = 0 // Disabled

	// Create a request with a timestamp 1 hour in the future
	req := model.EventRequest{
		EventName: "future_event", Channel: "web", UserID: "u1",
		Timestamp: s.service.now().Add(1 * time.Hour).Unix(),
	}

	_, err := s.service.BuildEvent(req)
	s.NoError(err, "Future timestamps should be allowed when tolerance is 0")
}

// TestProcessEvent verifies that valid events are properly enqueued to the worker.
func (s *EventServiceTestSuite) TestProcessEvent() {
	ctx := context.Background()
	event := model.Event{EventName: "click"}

	// Mock Expectation: Ensure the Enqueue method is called with the specific event
	s.worker.On("Enqueue", event).Return()

	s.service.ProcessEvent(ctx, event)

	// Assert that the mock method was actually called
	s.worker.AssertExpectations(s.T())
}

func (s *EventServiceTestSuite) TestGetMetrics_Validation() {
	_, err := s.service.GetMetrics(context.Background(), model.MetricsFilter{})
	s.Error(err)
	s.IsType(&ValidationError{}, err)
}

func (s *EventServiceTestSuite) TestGetMetrics_Success() {
	ctx := context.Background()
	now := time.Unix(2000, 0).UTC()
	s.service.now = func() time.Time { return now }

	filter := model.MetricsFilter{
		EventName: "signup",
	}
	expectedFilter := model.MetricsFilter{
		EventName: "signup",
		GroupBy:   "channel",
		To:        now,
		From:      now.Add(-30 * 24 * time.Hour),
	}

	groups := []model.MetricsGroup{{Key: "web", TotalCount: 8, UniqueUserCount: 2}}
	s.repo.On("FetchMetrics", mock.Anything, expectedFilter).Return(uint64(10), uint64(3), groups, nil)

	resp, err := s.service.GetMetrics(ctx, filter)

	s.NoError(err)
	s.Equal(uint64(10), resp.Data.TotalEventCount)
	s.Equal(uint64(3), resp.Data.UniqueEventCount)
	s.Equal("channel", resp.Meta.GroupBy)
	s.Equal(now.Add(-30*24*time.Hour).Format(time.RFC3339), resp.Meta.Period.Start)
	s.Equal(now.Format(time.RFC3339), resp.Meta.Period.End)
	s.Equal(groups, resp.Data.Groups)
}

func (s *EventServiceTestSuite) TestGetMetrics_InvalidGroupBy() {
	_, err := s.service.GetMetrics(context.Background(), model.MetricsFilter{EventName: "signup", GroupBy: "unknown"})
	s.Error(err)
	s.IsType(&ValidationError{}, err)
}

func (s *EventServiceTestSuite) TestGetMetrics_FromAfterTo() {
	from := time.Unix(20, 0).UTC()
	to := time.Unix(10, 0).UTC()
	_, err := s.service.GetMetrics(context.Background(), model.MetricsFilter{EventName: "signup", From: from, To: to})
	s.Error(err)
	s.IsType(&ValidationError{}, err)
}

// TestValidateTimestamp_Helper tests the standalone helper function logic.
func (s *EventServiceTestSuite) TestValidateTimestamp_Helper() {
	now := time.Unix(1000, 0)

	// Case 1: Within tolerance (Valid)
	err := ValidateTimestamp(now.Add(1*time.Second), now, 5*time.Second)
	s.NoError(err)

	// Case 2: Exceeds tolerance (Invalid)
	err = ValidateTimestamp(now.Add(10*time.Second), now, 5*time.Second)
	s.Error(err)

	// Case 3: Tolerance disabled (Valid)
	err = ValidateTimestamp(now.Add(100*time.Hour), now, 0)
	s.NoError(err)
}
