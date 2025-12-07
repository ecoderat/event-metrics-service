# Event Metrics Service

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![ClickHouse](https://img.shields.io/badge/ClickHouse-OLAP-FFCC01?style=flat&logo=clickhouse)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker)

`event-metrics-service` is a Go backend for **high-throughput event ingestion** and **near real-time metric aggregation**.

It demonstrates a simple but realistic architecture that uses a columnar OLAP store (**ClickHouse**) for analytics, and compares it against a traditional row-store (**PostgreSQL**) on the same workload.

---

## 🚀 Features

- **High-throughput ingestion**  
  HTTP `POST /events` endpoint + **in-memory queue** pattern to batch inserts.

- **OLAP storage in ClickHouse**  
  Events are stored in ClickHouse using a `ReplacingMergeTree` table tuned for append-only event data and deduplication.

- **On-the-fly metrics**  
  `GET /metrics` computes aggregates directly on ClickHouse (no pre-aggregated tables):  
  total events (`COUNT(*)`) and unique users (`COUNT(DISTINCT user_id)`).

- **Built-in load tester**  
  A dedicated Go tool (run as a container) to simulate heavy traffic and duplicate submissions.

---

## 🏗 Architecture

High-level data flow:

1. **API layer**  
   `POST /events` receives a single event JSON payload.

2. **Buffering**  
   The handler validates/parses the request and pushes events into a buffered Go channel (non-blocking from the client’s perspective).

3. **Batch processing**  
   Background workers drain the channel and insert batches into ClickHouse, reducing per-request overhead.

4. **Querying**  
   `GET /metrics` queries ClickHouse directly, aggregating over a requested time window and `event_name`.

For this assessment, metrics focus on **short/medium windows** (e.g. up to 7 days) to keep response times in the **sub-second to ~2s** range while still working directly off raw event data.

---

## 🛠 Getting Started

### Prerequisites

- Docker  
- Docker Compose

### Run the stack

Start the API and ClickHouse services. The database schema is applied automatically on startup.

```bash
docker-compose up --build
````

By default the API will listen on `http://localhost:8080`.

### Health check

```bash
curl http://localhost:8080/health
```

Expected:

```json
{ "status": "ok" }
```

---

## 🔌 API Reference

### 1. Ingest event

**POST** `/events`  
Accepts a single event payload.

#### Request

```bash
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_name": "product_view",
    "channel": "web",
    "campaign_id": "cmp_987",
    "user_id": "user_123",
    "timestamp": 1723475612,
    "tags": ["electronics", "homepage", "flash_sale"],
    "metadata": {
      "product_id": "prod-789",
      "price": 129.99,
      "currency": "TRY",
      "referrer": "google"
    }
  }'
````

#### Response

```text
202 Accepted
```

---

### 2. Get metrics

**GET** `/metrics`
Returns aggregated metrics for a given event type, over a time range, with optional filters and grouping.

#### Query parameters

* `event_name` (**required**)
  Name of the event to aggregate (e.g. `add_to_cart`, `product_view`).

* `channel` (optional)
  Filter by channel (e.g. `web`, `mobile_app`).
  If omitted, metrics are aggregated across **all channels**.

* `group_by` (optional, default: `channel`)
  Dimension used to group the results:

  * `channel`
  * `day`
  * `hour`

  If not provided, results are grouped by **channel**.

* `from` (optional)
  Start of the time range, as **Unix timestamp (seconds)**.

* `to` (optional)
  End of the time range, as **Unix timestamp (seconds)**.

  If **both** `from` and `to` are omitted, the service uses the **last 30 days** up to “now” as the time window.

#### Example request

```bash
curl "http://localhost:8080/metrics?event_name=add_to_cart&group_by=day&channel=web&from=1764450000&to=1764622800"
```

#### Example response

```json
{
  "meta": {
    "event_name": "add_to_cart",
    "period": {
      "start": "2025-11-29T21:00:00Z",
      "end": "2025-12-01T21:00:00Z"
    },
    "filters": {
      "channel": "web"
    },
    "group_by": "day"
  },
  "data": {
    "total_event_count": 122790,
    "unique_event_count": 39762,
    "groups": [
      {
        "key": "2025-11-29",
        "total_count": 7509,
        "unique_user_count": 6847
      },
      {
        "key": "2025-11-30",
        "total_count": 61511,
        "unique_user_count": 31465
      },
      {
        "key": "2025-12-01",
        "total_count": 53770,
        "unique_user_count": 29617
      }
    ]
  }
}
```

---

## ⚡ Benchmarking & Load Testing

A custom Go load tester is included and runs as a separate container in the same Docker network.

### Run a load test

```bash
# Ensure the stack is running
docker compose up --build -d

# Execute the load tester inside the cluster
docker compose exec load-tester sh -lc \
  'go run main.go \
    -endpoint http://app:8080/events \
    -total 20000 \
    -rate 2000 \
    -duplication-percent 20'
```

This scenario:

* Sends **20,000 events**
* Targets **2,000 req/s**
* Reuses **20%** of the payloads (`-duplication-percent`) to simulate duplicate submissions and exercise ClickHouse deduplication.

Use this to get a feel for ingestion throughput and basic stability on your machine.

---

## 📊 Performance & Database Choice

All benchmarks were run on a local **Apple M4** machine with **24 GB RAM**.

With ~**20M rows** in the `events` table, typical metrics queries (filter by `event_name` and time range, `COUNT(*)` + `COUNT(DISTINCT user_id)`) behave roughly as follows:

| ID     | Description                                         | Window  | PostgreSQL Exec Time | ClickHouse Exec Time |
| ------ | --------------------------------------------------- | ------- | -------------------- | -------------------- |
| q1-1d  | Hourly `product_view` metrics, grouped by `channel` | 1 day   | ~466 ms              | N/A                  |
| q1-7d  | Hourly `product_view` metrics, grouped by `channel` | 7 days  | ~1.96 s              | N/A                  |
| q1-30d | Hourly `product_view` metrics, grouped by `channel` | 30 days | ~7.77 s              | **≈180 ms**          |
| q2-7d  | Per-channel total events + unique users             | 7 days  | ~1.98 s              | **≈140 ms**          |
| q2-30d | Per-channel total events + unique users             | 30 days | N/A                  | **≈340 ms**          |
| q3-1d  | Daily metrics per `event_name` + unique users       | 1 day   | ~493 ms              | N/A                  |
| q3-7d  | Daily metrics per `event_name` + unique users       | 7 days  | ~2.02 s              | **≈130 ms**          |
| q3-30d | Daily metrics per `event_name` + unique users       | 30 days | N/A                  | **≈50 ms**           |

> **Summary:** Analytical queries that take **2–8 seconds** on PostgreSQL drop into the
> **50–340 ms** range on ClickHouse on the same dataset (roughly **10–30×** faster for this workload).

The exact SQL used for these benchmarks (for both PostgreSQL and ClickHouse) is kept in a separate file to keep this README compact:

* 👉 [`docs/queries.md`](docs/queries.md)

---

## 💡 Architectural Decisions

### Why PostgreSQL first, then ClickHouse?

As a baseline, the workload was first modelled on a single `events` table in PostgreSQL. This made it easy to:

* validate the event schema and HTTP API,
* experiment with idempotent ingestion using a simple
  `UNIQUE (idempotency_key)` + `INSERT ... ON CONFLICT DO NOTHING`,
* and understand how far a single relational table can go for mixed
  write-heavy + read-heavy analytics.

Once the dataset reached the millions of rows, PostgreSQL started to show limits for this pattern:

* Complex aggregations over 7–30 day windows took **2–8 seconds**.
* The same table served both **heavy inserts** and **read-intensive analytics**, increasing I/O and planning overhead.
* Row-oriented storage is not ideal for wide, append-only event streams with frequent `COUNT(DISTINCT ...)` and time-bucketing.

To explore a more scalable option, the same data and queries were ported to a `ReplacingMergeTree`-based `events` table in ClickHouse. On the same volume:

* Hourly `product_view` metrics for 30 days dropped from ~7.7 s to **≈180 ms**.
* Channel-level aggregates over 7 days dropped from ~2 s to **≈140 ms**.
* Daily event metrics over 30 days run in **≈50–130 ms**.

In this repository, ClickHouse is the **primary backing store** for events and metrics.
The PostgreSQL numbers are kept in the benchmark table as a realistic baseline for how a traditional row-store behaves under the same workload.

### Production evolution path

For a real production system, the design could evolve along these lines:

1. **Durable queue**
   Replace the in-memory queue with Kafka or RabbitMQ.  
   The API publishes events, a separate consumer batches them into ClickHouse, preventing data loss on crashes and enabling replay/backpressure.

3. **Operational vs. analytical split**
   Keep an RDBMS (e.g. PostgreSQL) as the **transactional system of record** if needed, while ClickHouse serves as the **analytics backend**.

4. **Caching hot queries**
   Add Redis (or similar) in front of `/metrics` for dashboard-style, repeated queries.

5. **Scaling out ClickHouse**
   Use sharded / replicated ClickHouse clusters once data grows into the billions of rows and multi-node setups are required.

This project keeps the implementation deliberately small and focused (single service + ClickHouse), while still showing a credible path to a production-grade, high-volume analytics stack.

