# Query samples

This document contains the SQL queries used for the benchmark table in the README.

- All queries operate on an `events` table with columns  
  `event_name`, `channel`, `campaign_id`, `user_id`, `ts`, `tags`, `metadata`, …
- The main metrics are:
  - `COUNT(*)` – total events
  - `COUNT(DISTINCT user_id)` / `countDistinct(user_id)` – unique users
- Time windows are usually **1 / 7 / 30 days**.  
  For brevity, some queries are shown with `30 days` and can be adapted by changing the interval.

---

## q1 – Hourly `product_view` metrics per channel

**Goal:**  
Per-hour metrics for the `product_view` event, grouped by `channel`  
→ benchmark IDs: `q1-1d`, `q1-7d`, `q1-30d`

### PostgreSQL

Generic shape (parameterised by `:window_days`):

```sql
SELECT
  DATE_TRUNC('hour', ts) AS metric_hour,
  channel,
  COUNT(*)               AS total_events,
  COUNT(DISTINCT user_id) AS unique_users
FROM events
WHERE event_name = 'product_view'
  AND ts >= NOW() - (:window_days || ' days')::interval
  AND ts <  NOW()
GROUP BY
  1, 2
ORDER BY
  1 DESC, 2;
````

Concrete 30-day variant used in the benchmark (`q1-30d`):

```sql
SELECT
  DATE_TRUNC('hour', ts) AS metric_hour,
  channel,
  COUNT(*)               AS total_events,
  COUNT(DISTINCT user_id) AS unique_users
FROM events
WHERE event_name = 'product_view'
  AND ts >= NOW() - INTERVAL '30 days'
  AND ts <  NOW()
GROUP BY
  1, 2
ORDER BY
  1 DESC, 2;
```

### ClickHouse

Concrete 30-day variant used in the benchmark (`q1-30d`):

```sql
SELECT
    toStartOfHour(ts)      AS metric_hour,
    channel,
    count()                AS total_events,
    countDistinct(user_id) AS unique_users
FROM events
WHERE event_name = 'product_view'
  AND ts >= now64(3, 'UTC') - INTERVAL 30 DAY
  AND ts <  now64(3, 'UTC')
GROUP BY
    metric_hour,
    channel
ORDER BY
    metric_hour DESC,
    channel;
```

To derive `q1-1d` or `q1-7d`, replace `INTERVAL 30 DAY` with `INTERVAL 1 DAY` or `INTERVAL 7 DAY`.

---

## q2 – Per-channel total events + unique users

**Goal:**
Aggregate by `channel` over a time window
→ benchmark IDs: `q2-7d`, `q2-30d`

### PostgreSQL

Generic version:

```sql
SELECT
  channel,
  COUNT(*)               AS total_events,
  COUNT(DISTINCT user_id) AS unique_users
FROM events
WHERE ts >= NOW() - (:window_days || ' days')::interval
  AND ts <  NOW()
GROUP BY
  1
ORDER BY
  total_events DESC;
```

7-day variant used in the benchmark (`q2-7d`):

```sql
SELECT
  channel,
  COUNT(*)               AS total_events,
  COUNT(DISTINCT user_id) AS unique_users
FROM events
WHERE ts >= NOW() - INTERVAL '7 days'
  AND ts <  NOW()
GROUP BY
  1
ORDER BY
  total_events DESC;
```

### ClickHouse

7-day variant (`q2-7d`):

```sql
SELECT
    channel,
    count()                AS total_events,
    countDistinct(user_id) AS unique_users
FROM events
WHERE ts >= now64(3, 'UTC') - INTERVAL 7 DAY
  AND ts <  now64(3, 'UTC')
GROUP BY
    channel
ORDER BY
    total_events DESC;
```

30-day variant (`q2-30d`):

```sql
SELECT
    channel,
    count()                AS total_events,
    countDistinct(user_id) AS unique_users
FROM events
WHERE ts >= now64(3, 'UTC') - INTERVAL 30 DAY
  AND ts <  now64(3, 'UTC')
GROUP BY
    channel
ORDER BY
    total_events DESC;
```

---

## q3 – Daily metrics per `event_name` + unique users

**Goal:**
Per-day, per-`event_name` metrics (total events + unique users)
→ benchmark IDs: `q3-1d`, `q3-7d`, `q3-30d`

### PostgreSQL

Generic version:

```sql
SELECT
  DATE_TRUNC('day', ts) AS metric_day,
  event_name,
  COUNT(*)               AS total_events,
  COUNT(DISTINCT user_id) AS unique_users
FROM events
WHERE ts >= NOW() - (:window_days || ' days')::interval
  AND ts <  NOW()
GROUP BY
  1, 2
ORDER BY
  1 DESC, 2;
```

7-day variant used in the benchmark (`q3-7d`):

```sql
SELECT
  DATE_TRUNC('day', ts) AS metric_day,
  event_name,
  COUNT(*)               AS total_events,
  COUNT(DISTINCT user_id) AS unique_users
FROM events
WHERE ts >= NOW() - INTERVAL '7 days'
  AND ts <  NOW()
GROUP BY
  1, 2
ORDER BY
  1 DESC, 2;
```

### ClickHouse

30-day variant (`q3-30d`):

```sql
SELECT
    toDate(ts)             AS metric_day,
    event_name,
    count()                AS total_events,
    countDistinct(user_id) AS unique_users
FROM events
WHERE ts >= now64(3, 'UTC') - INTERVAL 30 DAY
  AND ts <  now64(3, 'UTC')
GROUP BY
    metric_day,
    event_name
ORDER BY
    metric_day DESC,
    event_name;
```

7-day variant (`q3-7d`):

```sql
SELECT
    toDate(ts)             AS metric_day,
    event_name,
    count()                AS total_events,
    countDistinct(user_id) AS unique_users
FROM events
WHERE ts >= now64(3, 'UTC') - INTERVAL 7 DAY
  AND ts <  now64(3, 'UTC')
GROUP BY
    metric_day,
    event_name
ORDER BY
    metric_day DESC,
    event_name;
```

For a 1-day version (`q3-1d`), simply change the window to `INTERVAL 1 DAY`.

---

These queries are intentionally kept simple and unoptimised beyond basic filtering and grouping.
They are meant to illustrate the **shape** of the workload used in the benchmarks, not a fully tuned reporting layer.
