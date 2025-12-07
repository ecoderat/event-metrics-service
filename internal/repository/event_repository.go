package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"

	"event-metrics-service/internal/model"
)

// EventRepository defines database operations for events.
type EventRepository interface {
	// Create inserts a single event.
	Create(ctx context.Context, event model.Event) error

	// CreateBatch inserts multiple events efficiently using ClickHouse batches.
	CreateBatch(ctx context.Context, events []model.Event) error

	// FetchMetrics aggregates data based on filters.
	FetchMetrics(ctx context.Context, filter model.MetricsFilter) (int64, int64, []model.MetricsGroup, error)
}

type eventRepository struct {
	conn clickhouse.Conn
}

// NewEventRepository creates an EventRepository backed by ClickHouse.
func NewEventRepository(conn clickhouse.Conn) EventRepository {
	return &eventRepository{conn: conn}
}

const insertEventQuery = `
	INSERT INTO events (event_name, channel, campaign_id, user_id, ts, tags, metadata)
	VALUES (?, ?, ?, ?, ?, ?, ?)
`

func (r *eventRepository) Create(ctx context.Context, event model.Event) error {
	metadata, err := marshalMetadata(event.Metadata)
	if err != nil {
		return err
	}

	err = r.conn.Exec(ctx, insertEventQuery,
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

	batch, err := r.conn.PrepareBatch(ctx, insertEventQuery)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	for _, event := range events {
		metadata, err := marshalMetadata(event.Metadata)
		if err != nil {
			return err
		}

		if err := batch.Append(
			event.EventName,
			event.Channel,
			nullIfEmpty(event.CampaignID),
			event.UserID,
			event.Timestamp,
			event.Tags,
			metadata,
		); err != nil {
			return fmt.Errorf("append batch: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send batch: %w", err)
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

func marshalMetadata(metadata map[string]interface{}) (string, error) {
	if metadata == nil {
		return "{}", nil
	}
	b, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("marshal metadata: %w", err)
	}
	return string(b), nil
}

func nullIfEmpty(val string) interface{} {
	if val == "" {
		return nil
	}
	return val
}
