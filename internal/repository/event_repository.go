package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"event-metrics-service/internal/model"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// EventRepository defines database operations for events.
type EventRepository interface {
	// Create inserts a single event.
	Create(ctx context.Context, event model.Event) error

	// CreateBatch inserts multiple events efficiently using ClickHouse batches.
	CreateBatch(ctx context.Context, events []model.Event) error

	// FetchMetrics aggregates data based on filters.
	FetchMetrics(ctx context.Context, filter model.MetricsFilter) (uint64, uint64, []model.MetricsGroup, error)
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

func (r *eventRepository) FetchMetrics(ctx context.Context, filter model.MetricsFilter) (uint64, uint64, []model.MetricsGroup, error) {
	whereParts := []string{"event_name = ?"}
	args := []any{filter.EventName}

	if !filter.From.IsZero() {
		whereParts = append(whereParts, "ts >= ?")
		args = append(args, filter.From)
	}

	if !filter.To.IsZero() {
		whereParts = append(whereParts, "ts <= ?")
		args = append(args, filter.To)
	}

	if filter.Channel != nil && *filter.Channel != "" {
		whereParts = append(whereParts, "channel = ?")
		args = append(args, *filter.Channel)
	}

	where := ""
	if len(whereParts) > 0 {
		where = "WHERE " + strings.Join(whereParts, " AND ")
	}

	var totalCount, uniqueCount uint64
	totalsQuery := fmt.Sprintf("SELECT COUNT(*), COUNT(DISTINCT user_id) FROM events %s", where)
	row := r.conn.QueryRow(ctx, totalsQuery, args...)
	if err := row.Scan(&totalCount, &uniqueCount); err != nil {
		return 0, 0, nil, fmt.Errorf("query totals: %w", err)
	}

	groupQuery, err := buildGroupQuery(filter.GroupBy, where)
	if err != nil {
		return 0, 0, nil, err
	}

	rows, err := r.conn.Query(ctx, groupQuery, args...)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("query groups: %w", err)
	}
	defer rows.Close()

	groups, err := scanMetricGroups(rows)
	if err != nil {
		return 0, 0, nil, err
	}

	return totalCount, uniqueCount, groups, nil
}

func buildGroupQuery(groupBy, where string) (string, error) {
	// SQL injection protection: only allowed values are accepted via switch.
	switch groupBy {
	case "channel":
		return fmt.Sprintf(
			"SELECT channel, COUNT(*), COUNT(DISTINCT user_id) FROM events %s GROUP BY channel ORDER BY channel",
			where), nil
	case "hour":
		return fmt.Sprintf(
			"SELECT formatDateTime(ts, '%%Y-%%m-%%dT%%H:00:00Z'), COUNT(*), COUNT(DISTINCT user_id) FROM events %s GROUP BY 1 ORDER BY 1",
			where), nil
	case "day":
		return fmt.Sprintf(
			"SELECT formatDateTime(ts, '%%Y-%%m-%%d'), COUNT(*), COUNT(DISTINCT user_id) FROM events %s GROUP BY 1 ORDER BY 1",
			where), nil
	default:
		return "", fmt.Errorf("unsupported group_by: %s", groupBy)
	}
}

func scanMetricGroups(rows driver.Rows) ([]model.MetricsGroup, error) {
	var groups []model.MetricsGroup
	for rows.Next() {
		var key string
		var total, unique uint64
		if err := rows.Scan(&key, &total, &unique); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, model.MetricsGroup{
			Key:             key,
			TotalCount:      total,
			UniqueUserCount: unique,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate groups: %w", err)
	}
	return groups, nil
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
