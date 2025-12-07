# event-metrics-service
`event-metrics-service` is a Go backend service for **high-throughput event ingestion** and **metric aggregation**.

- Accepts events over HTTP (e.g. `POST /events`)
- Stores them in **ClickHouse** (ReplacingMergeTree)
- Uses an **in-memory queue + worker pool** to batch inserts
- Exposes aggregated metrics via `GET /metrics` (time range + event_name, total + unique users)

This version focuses on a simple but realistic design: near real-time inserts, on-the-fly metrics directly from ClickHouse.

---

## Run the app (Docker Compose)

Prerequisites:

- Docker
- docker-compose

Start the stack (API + ClickHouse):

```bash
docker-compose up --build
````

The service applies its ClickHouse schema on startup, so no separate migration step is required.

By default the API will listen on `http://localhost:8080`.

Basic sanity check:

```bash
curl http://localhost:8080/health
```

Expected:

```json
{ "status": "ok" }
```

---

## Benchmark (simple load test)

A small Go tool is included to send events to the API.

From the benchmark tool directory (e.g. `tools/bench/`):

```bash
go run main.go \
  -endpoint "events" \
  -total 20000 \
  -rate 2000
```

This will:

* Send **20,000** events
* At a target rate of **2,000 events/second**
* Against the `/events` endpoint on the running service

Use this to get a feel for ingestion throughput on your machine.

---

## Performance & database choice

The service now uses ClickHouse as the primary store so it can handle high insert rates and distinct counts efficiently. Events are written into a `ReplacingMergeTree` table keyed by `event_name`, `ts`, and `user_id`. Future hardening should add a durable queue (Kafka/Redpanda) ahead of ClickHouse, tune partitions/TTL for long-term retention, and add a small cache for hot metrics queries.
