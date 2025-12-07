package service

import (
	"context"
	"event-metrics-service/internal/model"
	"event-metrics-service/internal/repository"
	"log"
	"sync"
	"time"
)

type batchEventWorker struct {
	repo          repository.EventRepository // use repository for persistence
	eventQueue    chan model.Event
	batchSize     int
	flushInterval time.Duration
	wg            sync.WaitGroup
}

type BatchEventWorker interface {
	Enqueue(event model.Event)
	Shutdown()
}

// Constructor: inject repository instead of raw sql.DB
func NewbatchEventWorker(repo repository.EventRepository, bufferSize int, batchSize int, interval time.Duration) *batchEventWorker {
	worker := &batchEventWorker{
		repo:          repo, // dependency injection
		eventQueue:    make(chan model.Event, bufferSize),
		batchSize:     batchSize,
		flushInterval: interval,
	}
	worker.wg.Add(1)
	go worker.startLoop()
	return worker
}

// Enqueue receives events from the controller.
func (w *batchEventWorker) Enqueue(event model.Event) {
	// For high load, we could add a non-blocking send or drop policy; currently this blocks on a full buffer.
	w.eventQueue <- event
}

// Shutdown drains the queue and stops the worker.
func (w *batchEventWorker) Shutdown() {
	log.Println("Worker shutting down, waiting for queue to drain...")
	close(w.eventQueue)
	w.wg.Wait()
	log.Println("Worker shut down.")
}

// startLoop is the background worker loop.
func (w *batchEventWorker) startLoop() {
	defer w.wg.Done()

	var batch []model.Event
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-w.eventQueue:
			if !ok {
				log.Println("[INFO] Queue closed, flushing remaining events...")
				log.Println("[INFO] Event queue size: ", len(w.eventQueue))
				if len(batch) > 0 {
					w.bulkInsert(batch)
				}
				return
			}

			batch = append(batch, event)

			if len(batch) >= w.batchSize {
				log.Println("[INFO] batch size reached: ", len(batch))
				log.Println("[INFO] Event queue size: ", len(w.eventQueue))
				w.bulkInsert(batch)
				batch = nil
			}

		case <-ticker.C:
			log.Println("[INFO] timer tick, batch size: ", len(batch))
			log.Println("[INFO] Event queue size: ", len(w.eventQueue))
			if len(batch) > 0 {
				w.bulkInsert(batch)
				batch = nil
			}
		}
	}
}

func (w *batchEventWorker) bulkInsert(events []model.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := w.repo.CreateBatch(ctx, events)

	if err != nil {
		log.Printf("[ERROR] Bulk insert failed: %v", err)
	} else {
		log.Printf("[INFO] %d events flushed via repository", len(events))
	}
}
