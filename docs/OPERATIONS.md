# CQAR Operations Guide

Operational procedures, health checks, monitoring, and troubleshooting for CQAR in production.

## Table of Contents

- [Health Checks](#health-checks)
- [Metrics](#metrics)
- [Logging](#logging)
- [Tracing](#tracing)
- [Monitoring](#monitoring)
- [Alerting](#alerting)
- [Troubleshooting](#troubleshooting)
- [Maintenance](#maintenance)
- [Incident Response](#incident-response)

---

## Health Checks

CQAR exposes multiple health check endpoints for load balancer and orchestrator integration.

### Liveness Probe

**Endpoint**: `GET /health/live`  
**Port**: 8080 (HTTP)

**Purpose**: Indicates service process is running and responsive.

**Response** (Healthy):
```json
{
  "status": "ok"
}
```
HTTP 200

**Response** (Unhealthy):
```json
{
  "status": "error",
  "message": "service is shutting down"
}
```
HTTP 503

**Use Case**: Kubernetes liveness probe to restart unresponsive pods.

**Configuration**:
```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

---

### Readiness Probe

**Endpoint**: `GET /health/ready`  
**Port**: 8080 (HTTP)

**Purpose**: Indicates service is ready to accept traffic (dependencies healthy).

**Response** (Ready):
```json
{
  "status": "ready",
  "components": {
    "database": "ok",
    "cache": "ok",
    "eventbus": "ok"
  },
  "timestamp": "2025-10-16T12:34:56Z"
}
```
HTTP 200

**Response** (Not Ready):
```json
{
  "status": "not_ready",
  "components": {
    "database": "ok",
    "cache": "error: connection refused",
    "eventbus": "ok"
  },
  "timestamp": "2025-10-16T12:34:56Z"
}
```
HTTP 503

**Health Checks Performed**:
- **Database**: PostgreSQL connection pool health (`SELECT 1`)
- **Cache**: Redis ping command
- **Event Bus**: NATS connection status

**Use Case**: Kubernetes readiness probe to control traffic routing.

**Configuration**:
```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 2
```

---

### gRPC Health Check

**Service**: `grpc.health.v1.Health`  
**Method**: `Check`  
**Port**: 9090 (gRPC)

**Request**:
```bash
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

**Response** (Serving):
```json
{
  "status": "SERVING"
}
```

**Response** (Not Serving):
```json
{
  "status": "NOT_SERVING"
}
```

**Use Case**: gRPC-aware load balancers (Envoy, gRPC-LB).

---

## Metrics

CQAR exposes Prometheus metrics for monitoring and alerting.

### Metrics Endpoint

**Endpoint**: `GET /metrics`  
**Port**: 9091 (HTTP)  
**Format**: Prometheus text format

**Access**:
```bash
curl http://localhost:9091/metrics
```

---

### gRPC Metrics

#### Request Duration (Histogram)

**Metric**: `cqar_grpc_call_duration_seconds`  
**Type**: Histogram  
**Labels**: `method`, `status`

**Description**: gRPC method call latency distribution.

**Example**:
```
cqar_grpc_call_duration_seconds_bucket{method="/cqc.services.v1.AssetRegistry/GetAsset",status="OK",le="0.005"} 1543
cqar_grpc_call_duration_seconds_bucket{method="/cqc.services.v1.AssetRegistry/GetAsset",status="OK",le="0.01"} 1987
cqar_grpc_call_duration_seconds_bucket{method="/cqc.services.v1.AssetRegistry/GetAsset",status="OK",le="0.025"} 2134
cqar_grpc_call_duration_seconds_sum{method="/cqc.services.v1.AssetRegistry/GetAsset",status="OK"} 12.456
cqar_grpc_call_duration_seconds_count{method="/cqc.services.v1.AssetRegistry/GetAsset",status="OK"} 2150
```

**Query Examples**:
```promql
# p50 latency for GetAsset
histogram_quantile(0.50, rate(cqar_grpc_call_duration_seconds_bucket{method=~".*GetAsset"}[5m]))

# p99 latency for GetAsset
histogram_quantile(0.99, rate(cqar_grpc_call_duration_seconds_bucket{method=~".*GetAsset"}[5m]))

# Average latency by method
rate(cqar_grpc_call_duration_seconds_sum[5m]) / rate(cqar_grpc_call_duration_seconds_count[5m])
```

---

#### Request Count (Counter)

**Metric**: `cqar_grpc_calls_total`  
**Type**: Counter  
**Labels**: `method`, `status_code`

**Description**: Total gRPC calls by method and status.

**Example**:
```
cqar_grpc_calls_total{method="/cqc.services.v1.AssetRegistry/GetAsset",status_code="OK"} 15432
cqar_grpc_calls_total{method="/cqc.services.v1.AssetRegistry/GetAsset",status_code="NOT_FOUND"} 87
cqar_grpc_calls_total{method="/cqc.services.v1.AssetRegistry/CreateAsset",status_code="INVALID_ARGUMENT"} 23
```

**Query Examples**:
```promql
# Request rate by method
rate(cqar_grpc_calls_total[5m])

# Error rate
sum(rate(cqar_grpc_calls_total{status_code!="OK"}[5m])) / sum(rate(cqar_grpc_calls_total[5m]))

# Top 5 methods by call volume
topk(5, sum by (method) (rate(cqar_grpc_calls_total[5m])))
```

---

### Cache Metrics

#### Cache Hit Counter

**Metric**: `cqar_cache_hit_total`  
**Type**: Counter  
**Labels**: `entity`

**Description**: Cache hit count by entity type.

**Example**:
```
cqar_cache_hit_total{entity="asset"} 8765
cqar_cache_hit_total{entity="symbol"} 12543
cqar_cache_hit_total{entity="venue_symbol"} 45321
```

---

#### Cache Miss Counter

**Metric**: `cqar_cache_miss_total`  
**Type**: Counter  
**Labels**: `entity`

**Description**: Cache miss count by entity type.

**Example**:
```
cqar_cache_miss_total{entity="asset"} 432
cqar_cache_miss_total{entity="symbol"} 876
cqar_cache_miss_total{entity="venue_symbol"} 2341
```

**Query Examples**:
```promql
# Cache hit rate
sum(rate(cqar_cache_hit_total[5m])) / (sum(rate(cqar_cache_hit_total[5m])) + sum(rate(cqar_cache_miss_total[5m])))

# Cache hit rate by entity
sum by (entity) (rate(cqar_cache_hit_total[5m])) / (sum by (entity) (rate(cqar_cache_hit_total[5m])) + sum by (entity) (rate(cqar_cache_miss_total[5m])))
```

---

### Database Metrics

#### Query Duration (Histogram)

**Metric**: `cqar_db_query_duration_seconds`  
**Type**: Histogram  
**Labels**: `operation`

**Description**: Database query latency distribution.

**Example**:
```
cqar_db_query_duration_seconds_bucket{operation="GetAsset",le="0.01"} 1234
cqar_db_query_duration_seconds_bucket{operation="GetAsset",le="0.025"} 1456
cqar_db_query_duration_seconds_sum{operation="GetAsset"} 15.678
cqar_db_query_duration_seconds_count{operation="GetAsset"} 1500
```

**Query Examples**:
```promql
# p99 database query latency
histogram_quantile(0.99, rate(cqar_db_query_duration_seconds_bucket[5m]))

# Slow queries (>50ms)
cqar_db_query_duration_seconds_bucket{le="0.05"} - cqar_db_query_duration_seconds_bucket{le="0.025"}
```

---

### Event Bus Metrics

#### Events Published (Counter)

**Metric**: `cqar_event_published_total`  
**Type**: Counter  
**Labels**: `event_type`

**Description**: Total events published by type.

**Example**:
```
cqar_event_published_total{event_type="asset_created"} 543
cqar_event_published_total{event_type="symbol_created"} 234
cqar_event_published_total{event_type="quality_flag_raised"} 12
```

---

## Logging

CQAR uses structured JSON logging via zerolog.

### Log Levels

- **DEBUG**: Verbose debugging information (disabled in production)
- **INFO**: Normal operational messages (default in production)
- **WARN**: Warning conditions (degraded performance, deprecated usage)
- **ERROR**: Error conditions (failed requests, external service errors)
- **FATAL**: Critical errors causing service termination

### Log Format

```json
{
  "level": "info",
  "time": "2025-10-16T12:34:56.123Z",
  "service": "cqar",
  "version": "0.1.0",
  "request_id": "req-550e8400-e29b-41d4-a716-446655440000",
  "method": "/cqc.services.v1.AssetRegistry/GetAsset",
  "duration_ms": 12.45,
  "status": "OK",
  "asset_id": "btc-uuid",
  "cache_hit": true,
  "message": "request completed successfully"
}
```

### Log Fields

| Field         | Description        | Example                                   |
| ------------- | ------------------ | ----------------------------------------- |
| `level`       | Log level          | `info`, `error`                           |
| `time`        | ISO 8601 timestamp | `2025-10-16T12:34:56.123Z`                |
| `service`     | Service name       | `cqar`                                    |
| `version`     | Service version    | `0.1.0`                                   |
| `request_id`  | Unique request ID  | `req-550e8400...`                         |
| `method`      | gRPC method        | `/cqc.services.v1.AssetRegistry/GetAsset` |
| `duration_ms` | Request duration   | `12.45`                                   |
| `status`      | gRPC status code   | `OK`, `NOT_FOUND`                         |
| `error`       | Error message      | `asset not found`                         |
| `cache_hit`   | Cache hit status   | `true`, `false`                           |

### Configuration

**Log Level** (config.yaml):
```yaml
log:
  level: "info"        # debug, info, warn, error
  format: "json"       # json, text
  add_caller: false    # Add file:line to logs
```

**Environment Variable**:
```bash
export LOG_LEVEL=debug
./bin/cqar -config config.yaml
```

### Log Aggregation

**Recommended Setup**:
- **Fluent Bit**: Forward logs to Elasticsearch/Loki
- **Elasticsearch**: Index and search logs
- **Kibana**: Visualize and query logs

**Fluent Bit Configuration**:
```ini
[INPUT]
    Name              tail
    Path              /var/log/cqar/*.log
    Parser            json
    Tag               cqar

[OUTPUT]
    Name              es
    Match             cqar
    Host              elasticsearch.svc.cluster.local
    Port              9200
    Index             cqar-logs
    Type              _doc
```

---

## Tracing

CQAR integrates with OpenTelemetry for distributed tracing.

### Trace Configuration

**config.yaml**:
```yaml
tracing:
  enabled: true
  endpoint: "otel-collector:4317"
  sample_rate: 1.0        # 100% sampling (reduce in prod)
  service_name: "cqar"
```

### Trace Spans

**Automatic Spans**:
- gRPC method calls
- Database queries
- Cache operations
- Event publishing

**Span Attributes**:
- `service.name`: `cqar`
- `service.version`: `0.1.0`
- `rpc.method`: gRPC method name
- `db.statement`: SQL query
- `cache.key`: Cache key
- `event.type`: Event type

### Trace Backends

**Jaeger**:
```bash
# View traces
open http://localhost:16686
```

**Tempo** (Grafana):
```bash
# Query traces in Grafana
```

### Example Trace

```
Span: /cqc.services.v1.AssetRegistry/GetVenueSymbol (12ms)
├── Span: cache.get venue_symbol:binance:BTCUSDT (2ms) [HIT]
└── Span: enrichment.GetSymbol (8ms)
    ├── Span: cache.get symbol:btcusdt-spot-uuid (1ms) [HIT]
    └── [CACHED RESPONSE]
```

---

## Monitoring

### Grafana Dashboards

See [docs/monitoring/grafana-dashboard.json](monitoring/grafana-dashboard.json) for pre-built dashboard.

**Dashboard Panels**:
1. **Request Rate**: Requests/sec by method
2. **Request Latency**: p50, p90, p99 latency by method
3. **Error Rate**: Error percentage by method
4. **Cache Hit Rate**: Cache hit percentage by entity
5. **Database Query Latency**: p99 database query time
6. **Event Publishing Rate**: Events/sec by type
7. **Service Health**: Liveness/readiness status
8. **Infrastructure**: CPU, memory, network usage

**Access**:
```bash
# Import dashboard
curl -X POST http://grafana:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @docs/monitoring/grafana-dashboard.json
```

---

### Key Performance Indicators (KPIs)

| Metric                         | Target   | Critical Threshold | Alert |
| ------------------------------ | -------- | ------------------ | ----- |
| GetAsset p50 latency           | <10ms    | >20ms              | Yes   |
| GetAsset p99 latency           | <20ms    | >50ms              | Yes   |
| GetVenueSymbol p50 latency     | <10ms    | >20ms              | Yes   |
| GetVenueSymbol p99 latency     | <50ms    | >100ms             | Yes   |
| Cache hit rate                 | >80%     | <70%               | Yes   |
| Error rate                     | <0.1%    | >1%                | Yes   |
| Request rate                   | Baseline | 10x baseline       | Warn  |
| Database connection pool usage | <80%     | >90%               | Yes   |

---

## Alerting

### Prometheus Alert Rules

See [docs/monitoring/alert-rules.yml](monitoring/alert-rules.yml) for complete rules.

#### High Error Rate

```yaml
- alert: CQARHighErrorRate
  expr: |
    sum(rate(cqar_grpc_calls_total{status_code!="OK"}[5m]))
    / sum(rate(cqar_grpc_calls_total[5m])) > 0.01
  for: 5m
  labels:
    severity: critical
    service: cqar
  annotations:
    summary: "CQAR error rate above 1%"
    description: "Error rate is {{ $value | humanizePercentage }} (threshold: 1%)"
```

---

#### High Latency

```yaml
- alert: CQARHighLatency
  expr: |
    histogram_quantile(0.99,
      rate(cqar_grpc_call_duration_seconds_bucket{method=~".*Get.*"}[5m])
    ) > 0.050
  for: 5m
  labels:
    severity: warning
    service: cqar
  annotations:
    summary: "CQAR p99 latency above 50ms"
    description: "p99 latency is {{ $value | humanizeDuration }} for method {{ $labels.method }}"
```

---

#### Low Cache Hit Rate

```yaml
- alert: CQARLowCacheHitRate
  expr: |
    sum(rate(cqar_cache_hit_total[5m]))
    / (sum(rate(cqar_cache_hit_total[5m])) + sum(rate(cqar_cache_miss_total[5m])))
    < 0.70
  for: 10m
  labels:
    severity: warning
    service: cqar
  annotations:
    summary: "CQAR cache hit rate below 70%"
    description: "Cache hit rate is {{ $value | humanizePercentage }} (threshold: 70%)"
```

---

#### Database Connection Pool Exhaustion

```yaml
- alert: CQARDatabasePoolExhausted
  expr: |
    cqar_db_pool_active_connections / cqar_db_pool_max_connections > 0.90
  for: 5m
  labels:
    severity: critical
    service: cqar
  annotations:
    summary: "CQAR database pool usage above 90%"
    description: "Pool usage: {{ $value | humanizePercentage }}"
```

---

#### Service Down

```yaml
- alert: CQARServiceDown
  expr: up{job="cqar"} == 0
  for: 1m
  labels:
    severity: critical
    service: cqar
  annotations:
    summary: "CQAR service is down"
    description: "Service {{ $labels.instance }} is not responding"
```

---

## Troubleshooting

### Service Won't Start

**Symptom**: Service exits immediately after start.

**Diagnostics**:
```bash
# Check logs
docker logs cqar

# Check configuration
./bin/cqar -config config.yaml 2>&1 | grep -i error

# Verify infrastructure
docker-compose ps
```

**Common Causes**:

1. **Database Connection Failed**:
   ```
   ERROR: failed to connect to database: connection refused
   ```
   **Fix**: Verify PostgreSQL is running, check credentials in config.yaml

2. **Redis Connection Failed**:
   ```
   ERROR: failed to connect to cache: connection refused
   ```
   **Fix**: Verify Redis is running, check host/port in config.yaml

3. **Port Already in Use**:
   ```
   ERROR: failed to start gRPC server: bind: address already in use
   ```
   **Fix**: Check for existing process on port 9090
   ```bash
   lsof -i :9090
   kill <PID>
   ```

4. **Migration Not Applied**:
   ```
   ERROR: relation "assets" does not exist
   ```
   **Fix**: Run migrations
   ```bash
   make migrate-up
   ```

---

### High Latency

**Symptom**: GetAsset/GetVenueSymbol p99 latency >50ms.

**Diagnostics**:
```promql
# Check cache hit rate
sum(rate(cqar_cache_hit_total[5m])) / (sum(rate(cqar_cache_hit_total[5m])) + sum(rate(cqar_cache_miss_total[5m])))

# Check database query latency
histogram_quantile(0.99, rate(cqar_db_query_duration_seconds_bucket[5m]))

# Check CPU/memory usage
rate(process_cpu_seconds_total{job="cqar"}[5m])
process_resident_memory_bytes{job="cqar"}
```

**Common Causes**:

1. **Low Cache Hit Rate (<70%)**:
   - **Fix**: Increase cache TTL, warm cache on startup, scale Redis

2. **Slow Database Queries**:
   - **Fix**: Add indexes, optimize queries, check PostgreSQL slow query log
   ```sql
   -- Check slow queries
   SELECT query, mean_exec_time, calls
   FROM pg_stat_statements
   ORDER BY mean_exec_time DESC
   LIMIT 10;
   ```

3. **Database Connection Pool Exhausted**:
   - **Fix**: Increase `max_conns` in config.yaml
   ```yaml
   database:
     max_conns: 50  # Increase from 25
   ```

4. **High CPU Usage**:
   - **Fix**: Scale horizontally (add replicas)

---

### Cache Issues

**Symptom**: Cache hit rate drops suddenly, high miss rate.

**Diagnostics**:
```bash
# Check Redis health
redis-cli ping

# Check Redis memory
redis-cli info memory

# Check cache keys
redis-cli KEYS "asset:*" | wc -l
redis-cli KEYS "symbol:*" | wc -l
```

**Common Causes**:

1. **Redis Eviction**:
   ```bash
   redis-cli INFO stats | grep evicted_keys
   ```
   **Fix**: Increase Redis memory, adjust TTLs

2. **Redis Restart**:
   - **Fix**: Configure Redis persistence (RDB/AOF)

3. **Cache Key Pattern Change**:
   - **Fix**: Verify cache key format matches code

---

### Event Publishing Failures

**Symptom**: Events not appearing in NATS stream.

**Diagnostics**:
```bash
# Check NATS connection
nats account info

# Check stream
nats stream info cqc_events

# Check service logs
grep "event_published" /var/log/cqar/cqar.log
```

**Common Causes**:

1. **NATS Connection Lost**:
   ```
   ERROR: failed to publish event: connection closed
   ```
   **Fix**: Check NATS server status, verify network connectivity

2. **Stream Not Created**:
   ```
   ERROR: stream not found: cqc_events
   ```
   **Fix**: Create stream manually or via CQI bootstrap

---

### Memory Leak

**Symptom**: Memory usage increases over time without bound.

**Diagnostics**:
```bash
# Check memory usage
curl http://localhost:9091/metrics | grep process_resident_memory_bytes

# Profile heap
go tool pprof http://localhost:9091/debug/pprof/heap
```

**Common Causes**:

1. **Goroutine Leak**:
   ```bash
   # Check goroutine count
   curl http://localhost:9091/metrics | grep go_goroutines
   ```
   **Fix**: Review code for goroutines not terminated

2. **Cache Growth**:
   - **Fix**: Verify Redis TTLs are set, check eviction policy

---

## Maintenance

### Rolling Updates

**Zero-Downtime Deployment**:

1. Deploy new version with readiness probe
2. Wait for new pods to become ready
3. Terminate old pods gracefully (30s shutdown timeout)

**Kubernetes Example**:
```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1
```

---

### Database Migrations

**Apply Migrations**:
```bash
# Test migration (dry-run)
make migrate-dry-run

# Apply migration
make migrate-up

# Rollback if needed
make migrate-down
```

**Migration Safety**:
- Always test in staging first
- Migrations must be backward-compatible
- Avoid locking tables in production

---

### Cache Invalidation

**Manual Invalidation**:
```bash
# Invalidate asset cache
redis-cli DEL asset:<asset_id>

# Invalidate all assets
redis-cli KEYS "asset:*" | xargs redis-cli DEL

# Invalidate venue symbols
redis-cli KEYS "venue_symbol:*" | xargs redis-cli DEL
```

**Automatic Invalidation**:
- Updates automatically invalidate cache
- No manual invalidation needed for normal operations

---

### Backup and Recovery

**Database Backup**:
```bash
# Backup PostgreSQL
pg_dump -h localhost -U cqar cqar > cqar_backup_$(date +%Y%m%d).sql

# Restore backup
psql -h localhost -U cqar cqar < cqar_backup_20251016.sql
```

**Cache Backup**:
```bash
# Redis snapshot
redis-cli BGSAVE

# Copy RDB file
cp /var/lib/redis/dump.rdb /backup/redis_$(date +%Y%m%d).rdb
```

---

## Incident Response

### Incident Severity Levels

| Level             | Description                    | Response Time | Example                             |
| ----------------- | ------------------------------ | ------------- | ----------------------------------- |
| **P1 - Critical** | Service down, data loss        | <15 min       | Database unavailable, service crash |
| **P2 - High**     | Degraded performance           | <30 min       | High latency, error rate >1%        |
| **P3 - Medium**   | Partial functionality impacted | <2 hours      | Cache down, single method failing   |
| **P4 - Low**      | Minor issues, no user impact   | <24 hours     | Low cache hit rate, slow queries    |

---

### Incident Response Checklist

**P1 - Critical Incident**:

1. **Assess Impact** (0-5 min):
   - Check service health: `curl http://cqar:8080/health/ready`
   - Check metrics: Error rate, latency, availability
   - Identify affected users/services

2. **Mitigate** (5-15 min):
   - Rollback recent deployment if applicable
   - Scale horizontally if resource exhaustion
   - Failover to backup infrastructure if available

3. **Communicate** (ongoing):
   - Notify stakeholders via incident channel
   - Update status page
   - Post-mortem after resolution

4. **Root Cause Analysis** (post-incident):
   - Review logs, metrics, traces
   - Identify root cause
   - Implement preventive measures

---

### Runbooks

#### Runbook: Service Unavailable (503)

**Symptoms**: Health check returns 503, service not serving traffic.

**Checklist**:
1. Check infrastructure:
   ```bash
   docker-compose ps
   kubectl get pods -l app=cqar
   ```
2. Check database connectivity:
   ```bash
   psql -h <db_host> -U cqar -c "SELECT 1"
   ```
3. Check Redis connectivity:
   ```bash
   redis-cli -h <redis_host> PING
   ```
4. Check logs for errors:
   ```bash
   kubectl logs -l app=cqar --tail=100
   ```
5. Restart service if necessary:
   ```bash
   kubectl rollout restart deployment/cqar
   ```

---

#### Runbook: High Error Rate

**Symptoms**: Error rate >1%, alerts firing.

**Checklist**:
1. Identify failing methods:
   ```promql
   topk(5, sum by (method) (rate(cqar_grpc_calls_total{status_code!="OK"}[5m])))
   ```
2. Check error codes:
   ```promql
   sum by (status_code) (rate(cqar_grpc_calls_total{status_code!="OK"}[5m]))
   ```
3. Review logs for specific errors:
   ```bash
   kubectl logs -l app=cqar | grep ERROR | tail -50
   ```
4. Check dependencies (database, cache):
   - Database query errors?
   - Cache connection errors?
5. Rollback if related to recent deployment

---

**Related Documentation**:
- [API.md](API.md) - API reference
- [DEPLOYMENT.md](DEPLOYMENT.md) - Deployment procedures
- [README.md](../README.md) - Getting started
