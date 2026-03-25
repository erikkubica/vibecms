# VibeCMS Observability Strategy

## About This Document

**Purpose:** Metrics, logging, tracing, and alerting strategy so the team can understand system health and diagnose issues in production.

**How AI tools should use this:** Instrument new code with the logging patterns and metric names defined here; do not introduce new logging formats or metric namespaces that conflict with this document.

**Consistency requirements:** Service names and component boundaries must match architecture.md; SLO targets should derive from the performance goals in goals.md; log fields should reference the data model from database-schema.md where relevant.

VibeCMS is built for extreme performance, targeting a sub-50ms Time to First Byte (TTFB). Because each site is a standalone deployment managed by an agency, our observability strategy focuses on lightweight, structured instrumentation that can be easily aggregated by external monitoring dashboards. We prioritize identifying bottlenecks in the rendering pipeline, tracking the health of the embedded Tengo scripting engine, and ensuring that AI integrations and background tasks do not degrade the user experience.

---

### Service Level Objectives (SLOs)

| SLO | Target | Measurement Window | Error Budget |
|-----|--------|--------------------|-------------|
| Public Page TTFB (P95) | < 50ms | 30 days | 0.5% of requests may exceed |
| Admin UI Interaction Latency (P95) | < 100ms | 30 days | 1.0% of requests may exceed |
| API High Availability | > 99.9% | 30 days | ~43 mins of downtime allowed |
| Tengo Script Execution Success | > 99.0% | 7 days | 1% of scripts may timeout/fail |
| Backup Success Rate | 100% | 30 days | 0 failures allowed |

Every SLO defined above has a corresponding alert in the **Alerting Rules** section.

---

### Structured Logging

VibeCMS uses structured JSON logging to `stdout`. Logs must be parsable by standard collectors.

**Standard Format:** JSON
**Required Fields:**
- `t`: Timestamp (RFC3339Nano)
- `l`: Log Level (DEBUG, INFO, WARN, ERROR, FATAL)
- `svc`: Service name (e.g., `rendering_engine`, `tengo_vm`, `media_pipe`)
- `msg`: Human-readable message
- `tid`: Trace ID (correlates logs across a single request)
- `uid`: User ID (if authenticated)

**Log Levels:**
- **DEBUG:** Verbose execution details. *Example: "Tengo VM initialized for hook before_page_render"*
- **INFO:** Significant lifecycle events. *Example: "Published content node 102 (slug: /about-us)"*
- **WARN:** Non-fatal issues that may impact performance. *Example: "License check failed; entering soft-fail mode"*
- **ERROR:** Fatal within a request/task but app continues. *Example: "S3 upload failed for asset uuid: 550e8400..."*
- **FATAL:** Application cannot continue. *Example: "PostgreSQL connection refused on startup"*

**Mandatory Log Events:**
- **Request Lifecycle:** Method, Path, Status Code, Duration.
- **Tengo Execution:** Script path, Execution time, Exit status.
- **Database:** Query type, Table, Duration (if > 5ms).
- **External Calls:** Provider (OpenAI/Resend), Endpoint, Latency, Credits Used.

**Sensitive Fields (NEVER LOG):**
- Plaintext passwords (always masked `***`)
- `vibe.key` license signatures
- Session cookie values
- AI Provider API Keys

---

### Application Metrics

All metrics follow Prometheus naming conventions as utilized by the Fiber/Echo monitoring middleware.

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `vibe_http_request_duration_seconds` | Histogram | `method`, `path`, `status` | TTFB latency distribution |
| `vibe_http_requests_total` | Counter | `method`, `status` | Total volume of HTTP traffic |
| `vibe_tengo_script_duration_seconds` | Histogram | `script_path`, `hook` | Execution time for Tengo VM |
| `vibe_tengo_script_errors_total` | Counter | `script_path`, `error_type` | Count of script timeouts/panics |
| `vibe_db_query_duration_seconds` | Histogram | `operation`, `table` | Latency of GORM/Postgres operations |
| `vibe_ai_request_duration_seconds` | Histogram | `provider`, `feature` | Latency of OpenAI/Anthropic calls |
| `vibe_cache_hit_ratio` | Gauge | `cache_name` | Hit rate for In-Memory routing/sitemaps |
| `vibe_backup_status` | Gauge | `target` | 1 for success, 0 for failure (last run) |

---

### Distributed Tracing

As VibeCMS is a single-binary deployment per site, tracing focuses on **In-Process Propagation**.
- **Standard:** W3C TraceContext (`traceparent` header).
- **Instrumentation:** All Fiber/Echo middlewares must extract or generate a Trace ID.
- **Propagation:** The Trace ID must be injected into:
    1. All log lines within that request.
    2. Headers for external API calls (Resend/AI Providers).
    3. The Tengo VM context for script-level logging.
- **Critical Spans:** `RouteMatch`, `TengoExecution`, `PostgresQuery`, `JetRender`, `S3Upload`.

---

### Dashboards

1. **Vibe Health Monitoring (Agency View)**
    - Audience: On-call engineer/Agency manager.
    - Panels: Current TTFB (P95), Error Rate %, DB Connection Pool saturation, License status indicator.
    - Refresh: 60s.

2. **Rendering Pipeline Performance**
    - Audience: Developers.
    - Panels: Jet Render time vs. Tengo Execution time, Cache Hit % (Route Trie), Template missing warnings (log query).
    - Refresh: 5m.

3. **External Services & Tasks**
    - Audience: Agency manager.
    - Panels: AI Provider Latency, Mail delivery success (Counter), Backup completion timeline, S3 latency.
    - Refresh: 15m.

---

### Alerting Rules

| Alert Name | Condition | Severity | Response |
|------------|-----------|----------|----------|
| **LatencyCrisis** | `vibe_http_request_duration_seconds{p95} > 50ms` for 5 min | P1 | Investigate slow Tengo scripts or DB locks. |
| **HighErrorRate** | `vibe_http_requests_total{status=~"5.."}` > 2% for 2 min | P0 | Check Postgres connectivity and storage availability. |
| **LicenseSoftFail** | `vibe_license_status == 0` | P2 | Contact client for license renewal; AI features are disabled. |
| **BackupFailed** | `vibe_backup_status == 0` | P1 | Manually trigger backup and check S3 credentials. |
| **EmailDeliveryFailure** | `vibe_mail_errors_total > 5` in 10 min | P2 | Check Resend.com API status or SMTP credentials. |
| **TengoLoopDetected** | `vibe_tengo_script_errors_total{type="timeout"} > 0` | P1 | Locate and optimize the infinite loop in Tengo script. |

**Response Actions:**
- **P0/P1:** Immediate notification via Agency Dashboard API or Webhook (PagerDuty/Slack).
- **P2:** Non-urgent log entry in "Pending Actions" section of the Admin UI.