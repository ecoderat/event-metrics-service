# event-metrics-service
`event-metrics-service` is a Go backend service for **high-throughput event ingestion** and **metric aggregation**.

- Accepts events over HTTP (e.g. `POST /events`)
- Stores them in **PostgreSQL**
- Uses an **in-memory queue + worker pool** to insert into the DB
- Exposes aggregated metrics via `GET /metrics` (time range + event_name, total + unique users)

This version focuses on a simple but realistic design: near real-time inserts, on-the-fly metrics from PostgreSQL.

---

## Run the app (Docker Compose)

Prerequisites:

- Docker
- docker-compose

Start the stack (API + PostgreSQL):

```bash
docker-compose up --build
````

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

This service runs on a local Apple M4 machine with **24 GB RAM**.  
With ~**20M rows** in the `events` table, typical metrics queries (filter by `event_name` and time range, `COUNT(*)` + `COUNT(DISTINCT user_id)`) behave roughly as follows:

| Time range | Query shape                              | Exec time (approx.) |
| ---------- | ---------------------------------------- | ------------------- |
| 1 day      | total + distinct users                   | ~0.5 s              |
| 7 days     | total + distinct + hourly buckets        | ~2.0 s              |
| 30 days    | total + distinct (optionally grouped)    | ~7–8 s              |

For this assignment I scope the HTTP API to **short / medium windows** (e.g. up to 7 days).  
This keeps `/metrics` usable (sub-second to ~2s) while being explicit that metrics are **not strictly real-time**, which is allowed by the requirements.

From a pure analytics perspective, an OLAP engine like **ClickHouse** would be a better long-term fit for:

- wide time ranges (30–180+ days),
- heavy `COUNT(DISTINCT user_id)` and time-bucketed aggregations over billions of rows.

However, for this 48h assessment I chose **PostgreSQL** because:

- I can use a **simple `UNIQUE (idempotency_key)` constraint** and `INSERT ... ON CONFLICT DO NOTHING` for idempotent ingestion.
- The requirements explicitly say ingestion should be near real-time, but metrics don’t need to be fully real-time.
- Keeping a single storage backend avoids introducing extra components (e.g. Redis + ClickHouse + queue) and complex consistency rules just to emulate uniqueness.

In a production system, I would evolve this design to:

- put a **durable queue** (e.g. Kafka) between the API and storage,
- keep PostgreSQL as the **operational source of truth**,
- stream the same events into **ClickHouse** for large-scale analytics,
- and optionally add **Redis** as a cache in front of the metrics API for hot queries.
