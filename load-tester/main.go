package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	Endpoint           string
	Total              int
	Rate               int
	Concurrency        int
	DuplicationPercent int
}

func parseFlags() *Config {
	c := &Config{}
	flag.StringVar(&c.Endpoint, "endpoint", "", "Target URL (required)")
	flag.IntVar(&c.Total, "total", 10000, "Total requests")
	flag.IntVar(&c.Rate, "rate", 2000, "Requests per second")
	flag.IntVar(&c.Concurrency, "concurrency", 0, "Worker count (0=auto)")
	flag.IntVar(&c.DuplicationPercent, "duplication-percent", 0, "Duplication percent (0 = no duplicates)")
	flag.Parse()

	if c.Endpoint == "" {
		fmt.Fprintln(os.Stderr, "Error: -endpoint is required")
		flag.Usage()
		os.Exit(1)
	}

	if c.Concurrency == 0 {
		c.Concurrency = c.Rate / 20 // Auto-scale workers
		if c.Concurrency < 50 {
			c.Concurrency = 50
		}
	}

	if c.DuplicationPercent > 100 {
		c.DuplicationPercent = 100
	} else if c.DuplicationPercent < 0 {
		c.DuplicationPercent = 0
	}

	return c
}

type Stats struct {
	ok      uint64
	errors  uint64
	latency int64 // microseconds
}

type EventPool struct {
	mu  sync.RWMutex
	buf []map[string]any
	max int
}

func NewEventPool(max int) *EventPool {
	return &EventPool{buf: make([]map[string]any, 0, max), max: max}
}

func (p *EventPool) Add(evt map[string]any) {
	clone := cloneEvent(evt)
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.buf) >= p.max {
		p.buf = p.buf[1:]
	}
	p.buf = append(p.buf, clone)
}

func (p *EventPool) GetRandom(rng *rand.Rand) (map[string]any, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.buf) == 0 {
		return nil, false
	}
	idx := rng.Intn(len(p.buf))
	return cloneEvent(p.buf[idx]), true
}

func (s *Stats) AddOK(duration time.Duration) {
	atomic.AddUint64(&s.ok, 1)
	atomic.AddInt64(&s.latency, duration.Microseconds())
}

func (s *Stats) AddError() {
	atomic.AddUint64(&s.errors, 1)
}

func (s *Stats) StartLogger(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastOK, lastErr uint64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ok := atomic.LoadUint64(&s.ok)
			errs := atomic.LoadUint64(&s.errors)
			latTotal := atomic.LoadInt64(&s.latency)

			curOK := ok - lastOK
			curErr := errs - lastErr
			lastOK, lastErr = ok, errs

			avgLat := 0.0
			if ok > 0 {
				avgLat = float64(latTotal) / float64(ok) / 1000.0
			}

			log.Printf("[STATS] 1s -> OK: %d | ERR: %d | AvgLat: %.2fms | Total OK: %d", curOK, curErr, avgLat, ok)
		}
	}
}

func main() {
	cfg := parseFlags()
	stats := &Stats{}
	pool := NewEventPool(10000)

	// High-performance HTTP Client
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.Concurrency,
			MaxIdleConnsPerHost: cfg.Concurrency, // Critical: Keep as many connections open as there are workers.
			IdleConnTimeout:     90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

	log.Printf("Starting Load Test: Target=%s Rate=%d/s Total=%d Workers=%d", cfg.Endpoint, cfg.Rate, cfg.Total, cfg.Concurrency)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Stats Logger
	go stats.StartLogger(ctx)

	// Job Queue
	jobs := make(chan struct{}, cfg.Rate*2)
	var wg sync.WaitGroup
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rngs := make([]*rand.Rand, cfg.Concurrency)
	for i := 0; i < cfg.Concurrency; i++ {
		rngs[i] = rand.New(rand.NewSource(rng.Int63()))
	}

	// Workers
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go startWorker(client, cfg.Endpoint, jobs, stats, pool, cfg.DuplicationPercent, rngs[i], &wg)
	}

	// Rate Limiter (Main Loop)
	remaining := cfg.Total
	for remaining > 0 {
		start := time.Now()
		batch := cfg.Rate
		if remaining < batch {
			batch = remaining
		}

		for i := 0; i < batch; i++ {
			jobs <- struct{}{}
		}
		remaining -= batch

		elapsed := time.Since(start)
		if elapsed < time.Second {
			time.Sleep(time.Second - elapsed)
		}
	}

	close(jobs)
	wg.Wait()

	log.Printf("DONE. Total OK: %d | Total Errors: %d", atomic.LoadUint64(&stats.ok), atomic.LoadUint64(&stats.errors))
}

func startWorker(client *http.Client, endpoint string, jobs <-chan struct{}, stats *Stats, pool *EventPool, dupPercent int, rng *rand.Rand, wg *sync.WaitGroup) {
	defer wg.Done()

	headers := http.Header{"Content-Type": []string{"application/json"}}

	for range jobs {
		event := pickEvent(rng, pool, dupPercent)
		start := time.Now()

		err := sendEvent(client, endpoint, event, headers)
		if err != nil {
			stats.AddError()
			// Optional: Log the error
			// log.Printf("Error: %v", err)
		} else {
			stats.AddOK(time.Since(start))
		}
	}
}

func sendEvent(client *http.Client, url string, data any, headers http.Header) error {
	body, _ := json.Marshal(data)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	// Performance Hack: Read and discard the Body so the connection can be reused (Keep-Alive)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("http status: %d", resp.StatusCode)
	}
	return nil
}

var (
	eventNames = []string{"product_view", "add_to_cart", "checkout_start", "purchase"}
	channels   = []string{"web", "mobile_app", "api"}
	currencies = []string{"TRY", "USD", "EUR"}
)

func pickEvent(rng *rand.Rand, pool *EventPool, dupPercent int) map[string]any {
	if dupPercent > 0 && rng.Intn(100) < dupPercent {
		if evt, ok := pool.GetRandom(rng); ok {
			return evt
		}
	}
	evt := generateRandomEvent(rng)
	pool.Add(evt)
	return evt
}

func generateRandomEvent(rng *rand.Rand) map[string]any {
	return map[string]any{
		"event_name":  eventNames[rng.Intn(len(eventNames))],
		"channel":     channels[rng.Intn(len(channels))],
		"campaign_id": fmt.Sprintf("cmp_%03d", rng.Intn(100)),
		"user_id":     fmt.Sprintf("user_%d", rng.Intn(100000)),
		"timestamp":   time.Now().Unix() - int64(rng.Intn(60)), // Last 60 seconds
		"tags":        []string{"electronics", "sale"},
		"metadata": map[string]any{
			"price":    rng.Float64() * 100,
			"currency": currencies[rng.Intn(len(currencies))],
		},
	}
}

func cloneEvent(evt map[string]any) map[string]any {
	if evt == nil {
		return nil
	}
	clone := make(map[string]any, len(evt))
	for k, v := range evt {
		switch val := v.(type) {
		case []string:
			cpy := append([]string(nil), val...)
			clone[k] = cpy
		case map[string]any:
			nested := make(map[string]any, len(val))
			for nk, nv := range val {
				nested[nk] = nv
			}
			clone[k] = nested
		default:
			clone[k] = v
		}
	}
	return clone
}
