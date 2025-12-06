package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"event-metrics-service/internal/model"
)

// EventRepository defines database operations for events.
type EventRepository interface {
	// Create inserts a single event.
	Create(ctx context.Context, event model.Event) error

	// CreateBatch inserts multiple events efficiently using pgx.Batch.
	CreateBatch(ctx context.Context, events []model.Event) error

	// FetchMetrics aggregates data based on filters.
	FetchMetrics(ctx context.Context, filter model.MetricsFilter) (int64, int64, []model.MetricsGroup, error)
}

type eventRepository struct {
	pool *pgxpool.Pool
}

// NewEventRepository creates an EventRepository backed by PostgreSQL.
func NewEventRepository(pool *pgxpool.Pool) EventRepository {
	return &eventRepository{pool: pool}
}

const insertEventQuery = `
	INSERT INTO events (idempotency_key, event_name, channel, campaign_id, user_id, ts, tags, metadata)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (idempotency_key) DO NOTHING
`

func (r *eventRepository) Create(ctx context.Context, event model.Event) error {
	metadata, err := marshalMetadata(event.Metadata)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, insertEventQuery,
		event.IdempotencyKey,
		event.EventName,
		event.Channel,
		nullIfEmpty(event.CampaignID),
		event.UserID,
		event.Timestamp,
		event.Tags,
		metadata,
	)

	return err
}

func (r *eventRepository) CreateBatch(ctx context.Context, events []model.Event) error {
	if len(events) == 0 {
		return nil
	}

	batch := &pgx.Batch{}

	query := `
		INSERT INTO events (idempotency_key, event_name, channel, campaign_id, user_id, ts, tags, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (idempotency_key) DO NOTHING
	`

	for _, event := range events {
		metadata, err := marshalMetadata(event.Metadata)
		if err != nil {
			return err
		}

		batch.Queue(query,
			event.IdempotencyKey,
			event.EventName,
			event.Channel,
			nullIfEmpty(event.CampaignID),
			event.UserID,
			event.Timestamp,
			event.Tags,
			metadata,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range events {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("batch execution error: %w", err)
		}
	}

	return nil
}

func (r *eventRepository) FetchMetrics(ctx context.Context, filter model.MetricsFilter) (int64, int64, []model.MetricsGroup, error) {
	return 0, 0, nil, nil
}

func buildGroupQuery(groupBy, where string) (string, error) {
	// SQL Injection koruması: Sadece izin verilen değerleri switch ile alıyoruz.
	switch groupBy {
	case "channel":
		return fmt.Sprintf(
			"SELECT channel, COUNT(*), COUNT(DISTINCT user_id) FROM events %s GROUP BY channel ORDER BY channel",
			where), nil
	case "hour":
		return fmt.Sprintf(
			"SELECT to_char(to_timestamp(ts), 'YYYY-MM-DD\"T\"HH24:00:00\"Z\"'), COUNT(*), COUNT(DISTINCT user_id) FROM events %s GROUP BY 1 ORDER BY 1",
			where), nil
	case "day":
		return fmt.Sprintf(
			"SELECT to_char(to_timestamp(ts), 'YYYY-MM-DD'), COUNT(*), COUNT(DISTINCT user_id) FROM events %s GROUP BY 1 ORDER BY 1",
			where), nil
	default:
		return "", fmt.Errorf("unsupported group_by: %s", groupBy)
	}
}

func marshalMetadata(metadata map[string]interface{}) ([]byte, error) {
	if metadata == nil {
		return nil, nil // JSONB null
	}
	b, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	return b, nil
}

func nullIfEmpty(val string) interface{} {
	if val == "" {
		return nil
	}
	return val
}
