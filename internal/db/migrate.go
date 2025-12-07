package db

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// RunMigrations ensures required tables exist. This keeps the service
// self-contained without an external migration step.
func RunMigrations(ctx context.Context, conn clickhouse.Conn) error {
	err := conn.Exec(ctx, `
CREATE TABLE IF NOT EXISTS events
(
	event_name      String,
	channel         String,
	campaign_id     Nullable(String),
	user_id         String,
	ts              DateTime64(3, 'UTC'),
    tags            Array(String),
    metadata        String DEFAULT '{}',
	ingested_at     DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree
PARTITION BY toYYYYMMDD(ts)
ORDER BY (event_name, ts, user_id, channel, campaign_id)
SETTINGS
    allow_nullable_key = 1,
    index_granularity = 8192;
`)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
