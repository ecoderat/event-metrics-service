package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"event-metrics-service/internal/model"
	"event-metrics-service/internal/testdata/mockclickhousebatch"
	"event-metrics-service/internal/testdata/mockclickhouseconnection"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type EventRepositoryTestSuite struct {
	suite.Suite

	repository *eventRepository
	connMock   *mockclickhouseconnection.Connection
	batchMock  *mockclickhousebatch.Batch
}

func TestEventRepository(t *testing.T) {
	suite.Run(t, new(EventRepositoryTestSuite))
}

func (s *EventRepositoryTestSuite) SetupTest() {
	s.connMock = &mockclickhouseconnection.Connection{}
	s.batchMock = &mockclickhousebatch.Batch{}
	s.repository = &eventRepository{conn: s.connMock}
}

func (s *EventRepositoryTestSuite) TearDownTest() {
	s.connMock.AssertExpectations(s.T())
	s.batchMock.AssertExpectations(s.T())
}

func (s *EventRepositoryTestSuite) TestCreate_Success() {
	ctx := context.Background()
	ts := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	event := model.Event{
		EventName:  "product_view",
		Channel:    "web",
		CampaignID: "cmp-123",
		UserID:     "user-1",
		Timestamp:  ts,
		Tags:       []string{"electronics", "homepage"},
		Metadata: map[string]any{
			"price":  99.9,
			"source": "seo",
		},
	}

	metadataJSON, err := marshalMetadata(event.Metadata)
	s.Require().NoError(err)

	s.connMock.On(
		"Exec",
		mock.Anything,    // context
		insertEventQuery, // query
		event.EventName,  // event_name
		event.Channel,    // channel
		event.CampaignID, // nullIfEmpty -> string
		event.UserID,     // user_id
		event.Timestamp,  // ts
		event.Tags,       // tags
		metadataJSON,     // metadata (JSON string)
	).Return(nil).Once()

	err = s.repository.Create(ctx, event)
	s.NoError(err)
}

func (s *EventRepositoryTestSuite) TestCreate_NoCampaignID_UsesNil() {
	ctx := context.Background()
	ts := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	event := model.Event{
		EventName:  "purchase",
		Channel:    "mobile_app",
		CampaignID: "",
		UserID:     "user-2",
		Timestamp:  ts,
		Tags:       []string{"checkout"},
		Metadata:   nil,
	}

	metadataJSON, err := marshalMetadata(event.Metadata)
	s.Require().NoError(err)

	s.connMock.On(
		"Exec",
		mock.Anything,
		insertEventQuery,
		event.EventName,
		event.Channel,
		nil,
		event.UserID,
		event.Timestamp,
		event.Tags,
		metadataJSON,
	).Return(nil).Once()

	err = s.repository.Create(ctx, event)
	s.NoError(err)
}

func (s *EventRepositoryTestSuite) TestCreate_MetadataMarshalError() {
	ctx := context.Background()

	event := model.Event{
		EventName: "product_view",
		Channel:   "web",
		UserID:    "user-1",
		Timestamp: time.Now(),
		Tags:      []string{"test"},
		Metadata: map[string]any{
			"fn": func() {},
		},
	}

	err := s.repository.Create(ctx, event)
	s.Error(err)

	s.connMock.AssertNotCalled(s.T(), "Exec", mock.Anything, insertEventQuery, mock.Anything)
}

func (s *EventRepositoryTestSuite) TestCreateBatch_EmptySlice_NoOp() {
	ctx := context.Background()

	err := s.repository.CreateBatch(ctx, nil)
	s.NoError(err)

	err = s.repository.CreateBatch(ctx, []model.Event{})
	s.NoError(err)

	s.connMock.AssertNotCalled(s.T(), "PrepareBatch", mock.Anything, insertEventQuery)
	s.batchMock.AssertNotCalled(s.T(), "Append", mock.Anything)
	s.batchMock.AssertNotCalled(s.T(), "Send")
}

func (s *EventRepositoryTestSuite) TestCreateBatch_PrepareBatchError() {
	ctx := context.Background()

	events := []model.Event{
		{
			EventName: "product_view",
			Channel:   "web",
			UserID:    "user-1",
			Timestamp: time.Now(),
			Tags:      []string{"a"},
			Metadata:  map[string]any{"k": "v"},
		},
	}

	expectedErr := errors.New("prepare batch error")

	s.connMock.On(
		"PrepareBatch",
		mock.Anything,    // context
		insertEventQuery, // query
	).Return(nil, expectedErr).Once()

	err := s.repository.CreateBatch(ctx, events)

	s.ErrorIs(err, expectedErr)
	s.ErrorContains(err, "prepare batch")

	s.batchMock.AssertNotCalled(s.T(), "Append", mock.Anything)
	s.batchMock.AssertNotCalled(s.T(), "Send")
}

func (s *EventRepositoryTestSuite) TestCreateBatch_AppendError() {
	ctx := context.Background()

	events := []model.Event{
		{
			EventName:  "add_to_cart",
			Channel:    "web",
			CampaignID: "cmp-1",
			UserID:     "user-1",
			Timestamp:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			Tags:       []string{"cart"},
			Metadata:   map[string]any{"k": "v"},
		},
	}

	expectedErr := errors.New("append error")

	// PrepareBatch should return success.
	s.connMock.On(
		"PrepareBatch",
		mock.Anything,    // ctx
		insertEventQuery, // query
	).Return(s.batchMock, nil).Once()

	// Return error on Append call.
	s.batchMock.On(
		"Append",
		events[0].EventName,
		events[0].Channel,
		nullIfEmpty(events[0].CampaignID),
		events[0].UserID,
		events[0].Timestamp,
		events[0].Tags,
		mock.Anything, // metadata JSON string
	).Return(expectedErr).Once()

	err := s.repository.CreateBatch(ctx, events)

	s.ErrorIs(err, expectedErr)
	s.ErrorContains(err, "append batch")

	// Send should not be called when Append fails.
	s.batchMock.AssertNotCalled(s.T(), "Send")
}

func (s *EventRepositoryTestSuite) TestCreateBatch_SendError() {
	ctx := context.Background()

	events := []model.Event{
		{
			EventName:  "product_view",
			Channel:    "web",
			CampaignID: "cmp-1",
			UserID:     "user-1",
			Timestamp:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			Tags:       []string{"electronics"},
			Metadata:   map[string]any{"k": "v"},
		},
		{
			EventName:  "purchase",
			Channel:    "mobile_app",
			CampaignID: "", // nullIfEmpty → expects nil
			UserID:     "user-2",
			Timestamp:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			Tags:       []string{"checkout"},
			Metadata:   map[string]any{"step": "payment"},
		},
	}

	expectedErr := errors.New("send error")

	s.connMock.On(
		"PrepareBatch",
		mock.Anything,
		insertEventQuery,
	).Return(s.batchMock, nil).Once()

	// 1. event append success
	s.batchMock.On(
		"Append",
		events[0].EventName,
		events[0].Channel,
		nullIfEmpty(events[0].CampaignID),
		events[0].UserID,
		events[0].Timestamp,
		events[0].Tags,
		mock.Anything,
	).Return(nil).Once()

	// 2. event append success (CampaignID is empty → nil)
	s.batchMock.On(
		"Append",
		events[1].EventName,
		events[1].Channel,
		nullIfEmpty(events[1].CampaignID),
		events[1].UserID,
		events[1].Timestamp,
		events[1].Tags,
		mock.Anything,
	).Return(nil).Once()

	// Send returns error
	s.batchMock.On("Send").Return(expectedErr).Once()

	err := s.repository.CreateBatch(ctx, events)

	s.ErrorIs(err, expectedErr)
	s.ErrorContains(err, "send batch")
}

func (s *EventRepositoryTestSuite) TestCreateBatch_Success() {
	ctx := context.Background()

	events := []model.Event{
		{
			EventName:  "product_view",
			Channel:    "web",
			CampaignID: "cmp-1",
			UserID:     "user-1",
			Timestamp:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			Tags:       []string{"electronics"},
			Metadata:   map[string]any{"k": "v"},
		},
		{
			EventName:  "checkout_start",
			Channel:    "mobile_app",
			CampaignID: "", // expects nil to be passed
			UserID:     "user-2",
			Timestamp:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			Tags:       []string{"checkout"},
			Metadata:   map[string]any{"step": "payment"},
		},
	}

	s.connMock.On(
		"PrepareBatch",
		mock.Anything,
		insertEventQuery,
	).Return(s.batchMock, nil).Once()

	// 1. event append success
	s.batchMock.On(
		"Append",
		events[0].EventName,
		events[0].Channel,
		nullIfEmpty(events[0].CampaignID),
		events[0].UserID,
		events[0].Timestamp,
		events[0].Tags,
		mock.Anything,
	).Return(nil).Once()

	// 2. event append success
	s.batchMock.On(
		"Append",
		events[1].EventName,
		events[1].Channel,
		nullIfEmpty(events[1].CampaignID),
		events[1].UserID,
		events[1].Timestamp,
		events[1].Tags,
		mock.Anything,
	).Return(nil).Once()

	s.batchMock.On("Send").Return(nil).Once()

	err := s.repository.CreateBatch(ctx, events)
	s.NoError(err)
}
