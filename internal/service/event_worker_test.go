package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"event-metrics-service/internal/model"
	"event-metrics-service/internal/testdata/mockrepository"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type BatchWorkerTestSuite struct {
	suite.Suite
	mockRepo *mockrepository.Repository
	worker   *batchEventWorker
}

// TestBatchWorkerSuite is the entry point for the suite runner.
func TestBatchWorkerSuite(t *testing.T) {
	suite.Run(t, new(BatchWorkerTestSuite))
}

// SetupTest runs before each test method.
func (s *BatchWorkerTestSuite) SetupTest() {
	s.mockRepo = new(mockrepository.Repository)
}

// TearDownTest runs after each test method.
func (s *BatchWorkerTestSuite) TearDownTest() {
	s.mockRepo.AssertExpectations(s.T())
}

func (s *BatchWorkerTestSuite) TestBatchSizeTrigger() {
	// Configuration for this specific case
	batchSize := 5
	bufferSize := 10
	flushInterval := 1 * time.Hour // Long interval to prevent timer trigger

	// Synchronization: We use a WaitGroup to detect when the background worker calls the repo
	var wg sync.WaitGroup
	wg.Add(1)

	// Expectation: CreateBatch should be called exactly once with 5 events
	s.mockRepo.On("CreateBatch", mock.Anything, mock.MatchedBy(func(events []model.Event) bool {
		return len(events) == batchSize
	})).Run(func(args mock.Arguments) {
		wg.Done()
	}).Return(nil)

	// Initialize worker
	s.worker = NewbatchEventWorker(s.mockRepo, bufferSize, batchSize, flushInterval)
	defer s.worker.Shutdown() // Ensure cleanup

	// Action: Fill the batch
	for i := 0; i < batchSize; i++ {
		s.worker.Enqueue(model.Event{EventName: "test_event"})
	}

	// Assert: Wait for the async operation to complete
	s.waitForAsyncOp(&wg, "Batch Size Trigger")
}

func (s *BatchWorkerTestSuite) TestTimeIntervalTrigger() {
	// Configuration: Large batch size, but short interval
	batchSize := 10
	bufferSize := 10
	flushInterval := 50 * time.Millisecond

	var wg sync.WaitGroup
	wg.Add(1)

	// Expectation: A partial batch (3 events) should be flushed due to timer
	eventsToSend := 3
	s.mockRepo.On("CreateBatch", mock.Anything, mock.MatchedBy(func(events []model.Event) bool {
		return len(events) == eventsToSend
	})).Run(func(args mock.Arguments) {
		wg.Done()
	}).Return(nil)

	s.worker = NewbatchEventWorker(s.mockRepo, bufferSize, batchSize, flushInterval)
	defer s.worker.Shutdown()

	// Action: Send fewer events than batch size
	for i := 0; i < eventsToSend; i++ {
		s.worker.Enqueue(model.Event{EventName: "timed_event"})
	}

	// Assert: Wait for the timer to trigger the flush
	s.waitForAsyncOp(&wg, "Time Interval Trigger")
}

func (s *BatchWorkerTestSuite) TestShutdownFlush() {
	// Configuration
	batchSize := 10
	flushInterval := 1 * time.Hour

	// Expectation: Shutdown should flush whatever is in the queue
	eventsToSend := 4
	s.mockRepo.On("CreateBatch", mock.Anything, mock.MatchedBy(func(events []model.Event) bool {
		return len(events) == eventsToSend
	})).Return(nil)

	s.worker = NewbatchEventWorker(s.mockRepo, 10, batchSize, flushInterval)

	// Action: Enqueue items
	for i := 0; i < eventsToSend; i++ {
		s.worker.Enqueue(model.Event{EventName: "shutdown_event"})
	}

	// Action: Shutdown
	// This method blocks until the worker drains the queue, so we don't need a WaitGroup here.
	s.worker.Shutdown()

	// Assert: Verify mock was called
	s.mockRepo.AssertExpectations(s.T())
}

func (s *BatchWorkerTestSuite) TestGracefulErrorHandling() {
	// Configuration
	batchSize := 1
	flushInterval := 1 * time.Hour

	var wg sync.WaitGroup
	wg.Add(1)

	// Expectation: Repo returns an error (e.g., DB down), Worker should log it but not crash
	s.mockRepo.On("CreateBatch", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { wg.Done() }).
		Return(context.DeadlineExceeded)

	s.worker = NewbatchEventWorker(s.mockRepo, 10, batchSize, flushInterval)
	defer s.worker.Shutdown()

	s.worker.Enqueue(model.Event{EventName: "error_test"})

	// Assert: Wait for processing
	s.waitForAsyncOp(&wg, "Error Handling")

	// If the test reaches here without panicking, the worker handled the error gracefully.
	s.mockRepo.AssertExpectations(s.T())
}

// Helper method to wait for async operations with a timeout
func (s *BatchWorkerTestSuite) waitForAsyncOp(wg *sync.WaitGroup, testName string) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
		s.mockRepo.AssertExpectations(s.T())
	case <-time.After(1 * time.Second):
		s.T().Fatalf("Test '%s' timed out waiting for worker response", testName)
	}
}
