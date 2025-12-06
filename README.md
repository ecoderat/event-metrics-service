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

## What a production version would add

This v1 is intentionally simple. In a real high-scale environment, the architecture would typically evolve to:

* **Durable queue** between API and storage

  * API → Kafka/Redpanda
  * Consumers → PostgreSQL (operational store) and/or ClickHouse (analytics)
* **OLAP store for heavy metrics**

  * Use **ClickHouse** for large-scale aggregations (COUNT, DISTINCT, GROUP BY over billions of rows)
* **Cache layer**

  * Redis to cache popular metrics responses (short TTL)
  * Optionally use Redis for high-speed idempotency/dedup
* **Database hardening**

  * Time-based partitioning in PostgreSQL
  * Read replicas for separating ingestion and read load
  * More aggressive batching for inserts

This project focuses on a clean, understandable baseline implementation first; the production-grade ideas above are future steps, not all implemented here.
