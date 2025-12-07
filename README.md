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

### Query benchmark

All benchmarks were run on a local **Apple M4** machine with **24 GB RAM**.

With ~**20M rows** in the `events` table, typical metrics queries (filter by `event_name` and time range, `COUNT(*)` + `COUNT(DISTINCT user_id)`) behave roughly as follows:

| ID     | Description                                           | Window | PostgreSQL Exec Time | ClickHouse Exec Time |
|--------|-------------------------------------------------------|--------|----------------------|----------------------|
| q1-1d  | Hourly `product_view` metrics, grouped by `channel`   | 1 day  | ~466 ms              | N/A                  |
| q1-7d  | Hourly `product_view` metrics, grouped by `channel`   | 7 days | ~1.96 s              | N/A                  |
| q1-30d | Hourly `product_view` metrics, grouped by `channel`   | 30 days| ~7.77 s              | **≈180 ms**          |
| q2-7d  | Per-channel total events + unique users               | 7 days | ~1.98 s              | **≈140 ms**          |
| q2-30d | Per-channel total events + unique users               | 30 days| N/A                  | **≈340 ms**          |
| q3-1d  | Daily metrics per `event_name` + unique users         | 1 day  | ~493 ms              | N/A                  |
| q3-7d  | Daily metrics per `event_name` + unique users         | 7 days | ~2.02 s              | **≈130 ms**          |
| q3-30d | Daily metrics per `event_name` + unique users         | 30 days| N/A                  | **≈50 ms**           |

> In short, queries that take **2–8 seconds** on PostgreSQL drop into the
> **50–340 ms** range on ClickHouse on the same dataset  
> (roughly a **10–30× speed-up** for these workloads).

For this assignment, I scope the HTTP API to **short / medium time windows** (e.g. up to 7 days).  
This keeps `/metrics` responses in the **sub-second to ~2s** range on PostgreSQL, while being explicit that metrics are **not strictly real-time**, which is acceptable for the requirements.

From a pure analytics perspective, an OLAP engine like **ClickHouse** is a much better long-term fit for:

- wide time ranges (30–180+ days),
- heavy `COUNT(DISTINCT user_id)` and time-bucketed aggregations,
- and data volumes in the hundreds of millions / billions of rows.

### Why PostgreSQL first, then ClickHouse?

This assessment intentionally starts with a single `events` table in PostgreSQL.  
The first goal is to keep the design **as simple as possible**:

- ingest events via a JSON HTTP API,
- enforce idempotency with a plain `UNIQUE (idempotency_key)` constraint and  
  `INSERT ... ON CONFLICT DO NOTHING`,
- and run the required metrics queries directly on the same table.

As the dataset grows into the millions of rows, PostgreSQL begins to show limits for this pattern:

- Complex analytical queries over 7–30 day windows start taking **2–8 seconds**.
- The same table serves both **heavy inserts** and **read-intensive analytics**, increasing I/O and planning overhead.
- A row-store is not ideal for wide, read-heavy, append-only event data.

To explore the scaling path, I mirrored the same data into ClickHouse and ran the exact same queries on a `ReplacingMergeTree`-based `events` table. On the same volume:

- Hourly `product_view` metrics for 30 days dropped from ~7.7 s to **≈180 ms**.
- Channel-level aggregates over 7 days dropped from ~2 s to **≈140 ms**.
- Daily event metrics over 30 days run in **≈50–130 ms**.
