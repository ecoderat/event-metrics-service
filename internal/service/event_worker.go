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
	repo          repository.EventRepository // ARTIK REPO KULLANIYORUZ
	eventQueue    chan model.Event
	batchSize     int
	flushInterval time.Duration
	wg            sync.WaitGroup
}

type BatchEventWorker interface {
	Enqueue(event model.Event)
	Shutdown()
}

// Constructor değişti: sql.DB yerine repo alıyor
func NewbatchEventWorker(repo repository.EventRepository, bufferSize int, batchSize int, interval time.Duration) *batchEventWorker {
	worker := &batchEventWorker{
		repo:          repo, // Dependency Injection
		eventQueue:    make(chan model.Event, bufferSize),
		batchSize:     batchSize,
		flushInterval: interval,
	}
	worker.wg.Add(1)
	go worker.startLoop()
	return worker
}

// 4. Enqueue: Controller'ın çağıracağı metot
func (w *batchEventWorker) Enqueue(event model.Event) {
	// Non-blocking gönderim denenebilir veya doğrudan gönderilebilir.
	// Yük çok fazlaysa ve buffer dolduysa burada bir strateji belirlemek lazım.
	// Şimdilik standart gönderim yapıyoruz (Buffer dolarsa bloklar).
	w.eventQueue <- event
}

// 5. Shutdown: Graceful Shutdown için
func (w *batchEventWorker) Shutdown() {
	log.Println("Worker kapatılıyor, kanalın boşalması bekleniyor...")
	close(w.eventQueue) // Kanalı kapat, artık yeni veri gelmesin
	w.wg.Wait()         // startLoop bitene kadar bekle
	log.Println("Worker başarıyla kapatıldı.")
}

// 6. Internal Loop (Arka Plandaki İşçi)
func (w *batchEventWorker) startLoop() {
	defer w.wg.Done()

	var batch []model.Event
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-w.eventQueue:
			if !ok {
				// Kanal kapandı (Shutdown çağrıldı), elimizdekileri yazıp çıkalım
				log.Println("[INFO] Kanal kapandı, kalan veriler işleniyor...")
				log.Println("[INFO] Event queue size: ", len(w.eventQueue))
				if len(batch) > 0 {
					w.bulkInsert(batch)
				}
				return
			}

			batch = append(batch, event)

			// Batch boyutu doldu mu?
			if len(batch) >= w.batchSize {
				log.Println("[INFO] batch size reached: ", len(batch))
				log.Println("[INFO] Event queue size: ", len(w.eventQueue))
				w.bulkInsert(batch)
				batch = nil // veya batch = batch[:0] (memory optimizasyonu)
			}

		case <-ticker.C:
			// Süre doldu mu?
			log.Println("[INFO] timer tick, batch size: ", len(batch))
			log.Println("[INFO] Event queue size: ", len(w.eventQueue))
			if len(batch) > 0 {
				w.bulkInsert(batch)
				batch = nil
			}
		}
	}
}

// startLoop içindeki bulkInsert çağrısı değişti
func (w *batchEventWorker) bulkInsert(events []model.Event) {
	// Context oluştur
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// SQL yazmak yerine Repo metodunu çağırıyoruz
	// Worker'ın umrunda değil içeride Postgres mi var, Mock mu var, Console mu var.
	err := w.repo.CreateBatch(ctx, events)

	if err != nil {
		log.Printf("[ERROR] Bulk insert failed: %v", err)
	} else {
		log.Printf("[INFO] %d events flushed via repository", len(events))
	}
}
