# CQAR Monitoring Assets

Monitoring configuration files for CQAR observability.

## Files

- **alert-rules.yml**: Prometheus alerting rules for CQAR
- **grafana-dashboard.json**: Grafana dashboard for CQAR metrics visualization

## Prometheus Alert Rules

### Setup

**1. Add alert rules to Prometheus configuration:**

```yaml
# prometheus.yml
rule_files:
  - '/etc/prometheus/rules/cqar-alerts.yml'
```

**2. Copy alert rules file:**

```bash
kubectl create configmap prometheus-cqar-rules \
  --from-file=cqar-alerts.yml=alert-rules.yml \
  -n observability

# Mount in Prometheus deployment
volumes:
- name: cqar-rules
  configMap:
    name: prometheus-cqar-rules
volumeMounts:
- name: cqar-rules
  mountPath: /etc/prometheus/rules
```

**3. Reload Prometheus:**

```bash
# Send SIGHUP to Prometheus
kubectl exec -n observability prometheus-0 -- kill -HUP 1

# Or use HTTP API
curl -X POST http://prometheus:9090/-/reload
```

### Alert Summary

| Alert                         | Severity | Threshold            | Description                             |
| ----------------------------- | -------- | -------------------- | --------------------------------------- |
| CQARServiceDown               | Critical | 1 min                | Service not responding to health checks |
| CQARHighErrorRate             | Critical | >1% for 5 min        | Error rate exceeds threshold            |
| CQARHighLatencyGetAsset       | Warning  | p99 >50ms for 5 min  | GetAsset latency high                   |
| CQARHighLatencyGetVenueSymbol | Critical | p99 >100ms for 5 min | Critical for cqmd performance           |
| CQARLowCacheHitRate           | Warning  | <70% for 10 min      | Cache performance degraded              |
| CQARDatabasePoolExhausted     | Critical | >90% for 5 min       | Connection pool near limit              |
| CQARHighMemoryUsage           | Warning  | >85% for 5 min       | Memory usage high                       |
| CQARHighCPUUsage              | Warning  | >90% for 5 min       | CPU usage high                          |
| CQARRedisDown                 | Critical | 1 min                | Cache unavailable                       |
| CQARDatabaseDown              | Critical | 1 min                | Database unavailable                    |

### Alert Routing

**Example Alertmanager configuration:**

```yaml
route:
  group_by: ['alertname', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'platform-team'
  routes:
  - match:
      service: cqar
      severity: critical
    receiver: 'platform-pagerduty'
    continue: true
  - match:
      service: cqar
      severity: warning
    receiver: 'platform-slack'

receivers:
- name: 'platform-pagerduty'
  pagerduty_configs:
  - service_key: '<YOUR_PAGERDUTY_KEY>'
    description: '{{ .GroupLabels.alertname }}'

- name: 'platform-slack'
  slack_configs:
  - api_url: '<YOUR_SLACK_WEBHOOK>'
    channel: '#platform-alerts'
    title: 'CQAR Alert: {{ .GroupLabels.alertname }}'
    text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

---

## Grafana Dashboard

### Import Dashboard

**Option 1: Via Grafana UI**

1. Open Grafana
2. Navigate to **Dashboards** → **Import**
3. Upload `grafana-dashboard.json`
4. Select Prometheus data source
5. Click **Import**

**Option 2: Via API**

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <GRAFANA_API_KEY>" \
  -d @grafana-dashboard.json \
  http://grafana:3000/api/dashboards/db
```

**Option 3: Via ConfigMap (GitOps)**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cqar-dashboard
  namespace: observability
  labels:
    grafana_dashboard: "1"
data:
  cqar-dashboard.json: |
    <paste grafana-dashboard.json contents>
```

### Dashboard Panels

**Service Health**:
- Request Rate (req/s)
- Error Rate (%)
- Request Latency (p50, p90, p99)
- Service Health Status
- Pod Count

**Performance**:
- Cache Hit Rate by Entity
- Database Query Latency
- Database Connection Pool Usage
- Top Methods by Call Volume
- Top Methods by Error Rate

**Events**:
- Event Publishing Rate by Type

**Infrastructure**:
- CPU Usage
- Memory Usage
- Goroutines

### Dashboard Variables

Add variables for filtering:

```json
"templating": {
  "list": [
    {
      "name": "namespace",
      "type": "query",
      "query": "label_values(up{job=\"cqar\"}, namespace)",
      "current": {"text": "cqar", "value": "cqar"}
    },
    {
      "name": "pod",
      "type": "query",
      "query": "label_values(up{job=\"cqar\",namespace=\"$namespace\"}, pod)",
      "current": {"text": "All", "value": "$__all"}
    },
    {
      "name": "method",
      "type": "query",
      "query": "label_values(cqar_grpc_calls_total, method)",
      "current": {"text": "All", "value": "$__all"}
    }
  ]
}
```

---

## Monitoring Best Practices

### 1. Alert Fatigue Prevention

- **Tune thresholds**: Adjust based on baseline metrics
- **Use for clauses**: Require sustained violations before alerting
- **Group related alerts**: Group by service/component
- **Set priorities**: Critical vs Warning severity

### 2. Dashboard Organization

- **Service Overview**: Top-level KPIs (uptime, latency, errors)
- **Performance Deep-Dive**: Cache, database, event bus metrics
- **Infrastructure**: CPU, memory, network, disk
- **Business Metrics**: Asset counts, symbol counts, venue counts

### 3. SLO/SLI Tracking

Define Service Level Objectives:

| SLI            | SLO   | Measurement                                                             |
| -------------- | ----- | ----------------------------------------------------------------------- |
| Availability   | 99.9% | `up{job="cqar"}`                                                        |
| Latency (p99)  | <50ms | `cqar_grpc_call_duration_seconds`                                       |
| Error Rate     | <0.1% | `cqar_grpc_calls_total{status_code!="OK"}`                              |
| Cache Hit Rate | >80%  | `cqar_cache_hit_total / (cqar_cache_hit_total + cqar_cache_miss_total)` |

### 4. Alert Validation

Test alerts before production:

```bash
# Test alert expression
promtool check rules alert-rules.yml

# Query alert state
curl 'http://prometheus:9090/api/v1/alerts' | jq '.data.alerts[] | select(.labels.service == "cqar")'
```

---

## Runbook Links

Each alert includes a `runbook_url` annotation. Create runbooks at:

- https://docs.combine-capital.com/runbooks/cqar-service-down
- https://docs.combine-capital.com/runbooks/cqar-high-error-rate
- https://docs.combine-capital.com/runbooks/cqar-high-latency
- https://docs.combine-capital.com/runbooks/cqar-low-cache-hit-rate
- https://docs.combine-capital.com/runbooks/cqar-db-pool-exhausted
- https://docs.combine-capital.com/runbooks/redis-down
- https://docs.combine-capital.com/runbooks/postgres-down

Runbook template:

```markdown
# CQAR Service Down

## Severity: Critical

## Description
CQAR service is not responding to health checks.

## Impact
- cqmd cannot resolve venue symbols (price ingestion blocked)
- cqpm cannot fetch asset metadata (portfolio aggregation blocked)
- cqvx cannot map symbols for order execution

## Diagnosis
1. Check pod status: `kubectl get pods -n cqar`
2. Check logs: `kubectl logs -n cqar -l app=cqar --tail=100`
3. Check events: `kubectl get events -n cqar --sort-by='.lastTimestamp'`
4. Check dependencies: Database, Redis, NATS health

## Mitigation
1. If pod crash loop: Check logs for error, apply hotfix if needed
2. If database down: Restore database connection
3. If deployment issue: Rollback to previous version
4. If infrastructure: Scale up or migrate to healthy nodes

## Resolution
1. Fix root cause
2. Verify health: `curl http://cqar-http:8080/health/ready`
3. Update incident timeline
4. Schedule post-mortem
```

---

## Related Documentation

- [OPERATIONS.md](../OPERATIONS.md) - Operational procedures
- [DEPLOYMENT.md](../DEPLOYMENT.md) - Deployment guide
- [API.md](../API.md) - API reference
