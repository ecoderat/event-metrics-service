-- +migrate Up
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key VARCHAR(64) NOT NULL UNIQUE,
    event_name      VARCHAR(100) NOT NULL,
    channel         VARCHAR(50)  NOT NULL,
    campaign_id     VARCHAR(100),
    user_id         VARCHAR(100) NOT NULL,
    ts              TIMESTAMPTZ  NOT NULL,
    tags            TEXT[],
    metadata        JSONB,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_event_name_ts_channel
    ON events (event_name, ts, channel);

-- +migrate Down
DROP TABLE IF EXISTS events;
