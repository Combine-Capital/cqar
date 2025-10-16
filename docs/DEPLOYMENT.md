# CQAR Deployment Guide

Comprehensive deployment procedures for CQAR in production environments.

## Table of Contents

- [Infrastructure Requirements](#infrastructure-requirements)
- [Environment Variables](#environment-variables)
- [Configuration Management](#configuration-management)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Database Setup](#database-setup)
- [Security](#security)
- [Scaling](#scaling)
- [Disaster Recovery](#disaster-recovery)

---

## Infrastructure Requirements

### Compute Resources

**Service (per replica)**:
- **CPU**: 500m (request), 2000m (limit)
- **Memory**: 512Mi (request), 2Gi (limit)
- **Disk**: 10Gi for logs (ephemeral)

**Recommended Replicas**:
- **Development**: 1 replica
- **Staging**: 2 replicas
- **Production**: 3+ replicas (across availability zones)

---

### Dependencies

#### PostgreSQL 14+

**Requirements**:
- **Storage**: 100Gi+ SSD (depends on data volume)
- **CPU**: 4 cores minimum
- **Memory**: 8Gi minimum
- **Connections**: 100+ (25 per CQAR replica)
- **Backup**: Daily automated backups with 30-day retention

**Recommended Setup**:
- Managed service (AWS RDS, Google Cloud SQL, Azure Database)
- Multi-AZ deployment for high availability
- Read replicas for load distribution (future)

**Connection String**:
```
postgres://cqar_user:password@postgres.example.com:5432/cqar?sslmode=require
```

---

#### Redis 7+

**Requirements**:
- **Memory**: 4Gi minimum (cache size depends on dataset)
- **Persistence**: RDB snapshots or AOF for durability
- **Clustering**: Redis Cluster or Sentinel for HA

**Recommended Setup**:
- Managed service (AWS ElastiCache, Google Memorystore, Azure Cache)
- At least 3 nodes for high availability
- Eviction policy: `allkeys-lru` (least recently used)

**Connection String**:
```
redis://redis.example.com:6379/0
```

---

#### NATS JetStream 2.10+

**Requirements**:
- **Memory**: 2Gi minimum
- **Storage**: 50Gi for stream retention
- **Clustering**: 3+ nodes for quorum-based replication

**Recommended Setup**:
- Self-hosted on Kubernetes or managed service
- Stream retention: 7 days or 10GB
- Consumer acknowledgment: 30s timeout

**Connection String**:
```
nats://nats1.example.com:4222,nats2.example.com:4222,nats3.example.com:4222
```

---

### Network Requirements

**Ingress**:
- **gRPC**: Port 9090 (internal, load-balanced)
- **HTTP**: Port 8080 (health checks, internal only)
- **Metrics**: Port 9091 (Prometheus scraping, internal only)

**Egress**:
- PostgreSQL: Port 5432
- Redis: Port 6379
- NATS: Port 4222
- OpenTelemetry Collector: Port 4317 (if tracing enabled)

**Load Balancer**:
- gRPC-aware (Envoy, NGINX with gRPC support, Google Cloud Load Balancer)
- Health check: gRPC `grpc.health.v1.Health/Check`
- Connection draining: 30s

---

## Environment Variables

CQAR configuration can be overridden via environment variables.

### Core Configuration

```bash
# Service
SERVICE_NAME=cqar
SERVICE_VERSION=0.1.0
SERVICE_ENV=production

# Server
SERVER_HTTP_PORT=8080
SERVER_GRPC_PORT=9090
SERVER_SHUTDOWN_TIMEOUT=30s

# Database
DATABASE_HOST=postgres.example.com
DATABASE_PORT=5432
DATABASE_USER=cqar_prod_user
DATABASE_PASSWORD=${CQAR_DB_PASSWORD}  # From secret
DATABASE_DATABASE=cqar_prod
DATABASE_SSL_MODE=require
DATABASE_MAX_CONNS=25
DATABASE_MIN_CONNS=5
DATABASE_MAX_CONN_LIFETIME=15m
DATABASE_MAX_CONN_IDLE_TIME=5m
DATABASE_CONNECT_TIMEOUT=5s
DATABASE_QUERY_TIMEOUT=30s

# Cache (Redis)
CACHE_HOST=redis.example.com
CACHE_PORT=6379
CACHE_PASSWORD=${CQAR_REDIS_PASSWORD}  # From secret
CACHE_DB=0
CACHE_DEFAULT_TTL=60m
CACHE_MAX_RETRIES=3
CACHE_MIN_IDLE_CONNS=5
CACHE_POOL_SIZE=10

# Event Bus (NATS JetStream)
EVENTBUS_BACKEND=jetstream
EVENTBUS_SERVERS=nats://nats1:4222,nats://nats2:4222,nats://nats3:4222
EVENTBUS_STREAM_NAME=cqc_events
EVENTBUS_CONSUMER_NAME=cqar
EVENTBUS_MAX_DELIVER=3
EVENTBUS_ACK_WAIT=30s
EVENTBUS_MAX_ACK_PENDING=1000

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
LOG_ADD_CALLER=false

# Metrics
METRICS_ENABLED=true
METRICS_PORT=9091

# Tracing
TRACING_ENABLED=true
TRACING_ENDPOINT=otel-collector:4317
TRACING_SAMPLE_RATE=0.1  # 10% sampling in production

# Authentication
AUTH_API_KEYS=${CQAR_API_KEYS}  # Comma-separated: key1:name1,key2:name2
```

### Secret Management

**DO NOT HARDCODE SECRETS** in environment variables or config files.

**Recommended Approaches**:

1. **Kubernetes Secrets**:
   ```yaml
   env:
   - name: DATABASE_PASSWORD
     valueFrom:
       secretKeyRef:
         name: cqar-secrets
         key: database-password
   ```

2. **External Secrets Operator** (AWS Secrets Manager, HashiCorp Vault):
   ```yaml
   apiVersion: external-secrets.io/v1beta1
   kind: ExternalSecret
   metadata:
     name: cqar-secrets
   spec:
     secretStoreRef:
       name: aws-secrets-manager
     target:
       name: cqar-secrets
     data:
     - secretKey: database-password
       remoteRef:
         key: cqar/production/database
         property: password
   ```

3. **Environment Variables from Files** (Docker Swarm secrets):
   ```bash
   DATABASE_PASSWORD=$(cat /run/secrets/db_password)
   ```

---

## Configuration Management

### Configuration Priority

Override precedence (highest to lowest):
1. **Environment variables**
2. **config.prod.yaml** (environment-specific)
3. **config.yaml** (base configuration)

### Environment-Specific Configs

**config.yaml** (Base):
```yaml
service:
  name: "cqar"
  version: "0.1.0"

server:
  http_port: 8080
  grpc_port: 9090
  shutdown_timeout: "30s"

log:
  level: "info"
  format: "json"
```

**config.prod.yaml** (Production Overrides):
```yaml
service:
  env: "production"

database:
  max_conns: 50
  query_timeout: "30s"
  ssl_mode: "require"

cache:
  default_ttl: "60m"
  pool_size: 20

log:
  level: "info"
  add_caller: false

tracing:
  sample_rate: 0.1  # 10% sampling

auth:
  api_keys:
    # API keys loaded from secrets, not hardcoded
```

### ConfigMap (Kubernetes)

See [docs/k8s/configmap.yaml](k8s/configmap.yaml).

---

## Kubernetes Deployment

### Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: cqar
  labels:
    name: cqar
    environment: production
```

### Deployment

See [docs/k8s/deployment.yaml](k8s/deployment.yaml) for complete manifest.

**Key Configuration**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cqar
  namespace: cqar
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1
  selector:
    matchLabels:
      app: cqar
  template:
    metadata:
      labels:
        app: cqar
        version: v0.1.0
    spec:
      serviceAccountName: cqar
      containers:
      - name: cqar
        image: cqar:0.1.0
        ports:
        - name: grpc
          containerPort: 9090
        - name: http
          containerPort: 8080
        - name: metrics
          containerPort: 9091
        env:
        - name: SERVICE_ENV
          value: "production"
        envFrom:
        - configMapRef:
            name: cqar-config
        - secretRef:
            name: cqar-secrets
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 2Gi
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 2
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: cqar
              topologyKey: kubernetes.io/hostname
```

### Service

See [docs/k8s/service.yaml](k8s/service.yaml).

**gRPC Service**:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: cqar-grpc
  namespace: cqar
  annotations:
    cloud.google.com/neg: '{"ingress": true}'
spec:
  type: ClusterIP
  selector:
    app: cqar
  ports:
  - name: grpc
    port: 9090
    targetPort: 9090
    protocol: TCP
```

**HTTP Service (Metrics)**:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: cqar-metrics
  namespace: cqar
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9091"
spec:
  type: ClusterIP
  selector:
    app: cqar
  ports:
  - name: metrics
    port: 9091
    targetPort: 9091
    protocol: TCP
```

### Ingress / Load Balancer

**Google Cloud Load Balancer** (gRPC):
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cqar-ingress
  namespace: cqar
  annotations:
    kubernetes.io/ingress.class: "gce"
    kubernetes.io/ingress.allow-http: "false"
spec:
  rules:
  - host: cqar.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: cqar-grpc
            port:
              number: 9090
```

**Envoy Proxy** (Advanced routing):
See Envoy configuration in [docs/k8s/envoy-config.yaml](k8s/envoy-config.yaml).

---

### Deployment Steps

#### 1. Create Namespace

```bash
kubectl create namespace cqar
```

#### 2. Apply ConfigMap

```bash
kubectl apply -f docs/k8s/configmap.yaml
```

#### 3. Create Secrets

```bash
# Database password
kubectl create secret generic cqar-secrets \
  --from-literal=database-password='YOUR_DB_PASSWORD' \
  --from-literal=redis-password='YOUR_REDIS_PASSWORD' \
  --from-literal=api-keys='key1:service1,key2:service2' \
  -n cqar

# Or from external secrets manager
kubectl apply -f docs/k8s/external-secret.yaml
```

#### 4. Deploy Service

```bash
kubectl apply -f docs/k8s/deployment.yaml
kubectl apply -f docs/k8s/service.yaml
```

#### 5. Verify Deployment

```bash
# Check pods
kubectl get pods -n cqar

# Check logs
kubectl logs -n cqar -l app=cqar --tail=50

# Check health
kubectl port-forward -n cqar svc/cqar-grpc 9090:9090
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

#### 6. Apply Ingress

```bash
kubectl apply -f docs/k8s/ingress.yaml

# Wait for load balancer provisioning
kubectl get ingress -n cqar -w
```

---

## Database Setup

### Initial Setup

#### 1. Create Database

```sql
-- Connect as admin user
CREATE DATABASE cqar_prod;
CREATE USER cqar_prod_user WITH PASSWORD 'SECURE_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE cqar_prod TO cqar_prod_user;
```

#### 2. Run Migrations

**From Local Machine**:
```bash
# Set database URL
export DATABASE_URL="postgres://cqar_prod_user:SECURE_PASSWORD@postgres.example.com:5432/cqar_prod?sslmode=require"

# Run migrations
make migrate-up
```

**From Kubernetes Job**:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: cqar-migrate
  namespace: cqar
spec:
  template:
    spec:
      containers:
      - name: migrate
        image: cqar:0.1.0
        command: ["/bin/sh"]
        args:
        - -c
        - |
          migrate -path /app/migrations \
                  -database "$DATABASE_URL" \
                  up
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: cqar-secrets
              key: database-url
      restartPolicy: OnFailure
```

### Migration Best Practices

1. **Test in Staging**: Always test migrations in staging first
2. **Backward Compatible**: Ensure migrations don't break running services
3. **Rollback Plan**: Test rollback (`migrate down`) before production
4. **Backup**: Take database snapshot before migration
5. **Non-Blocking**: Avoid table locks (use `CONCURRENTLY` for indexes)

### Database Tuning

**postgresql.conf**:
```ini
# Connection pooling
max_connections = 200

# Memory
shared_buffers = 2GB
effective_cache_size = 6GB
work_mem = 16MB

# WAL
wal_buffers = 16MB
checkpoint_timeout = 15min
max_wal_size = 2GB

# Query optimizer
random_page_cost = 1.1  # SSD
effective_io_concurrency = 200

# Logging
log_min_duration_statement = 1000  # Log queries >1s
```

---

## Security

### Network Security

**Firewall Rules**:
- Only allow gRPC port (9090) from internal services
- Restrict metrics port (9091) to monitoring namespace
- Database/Redis accessible only from CQAR namespace

**Kubernetes Network Policies**:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: cqar-network-policy
  namespace: cqar
spec:
  podSelector:
    matchLabels:
      app: cqar
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: cqmd  # Allow cqmd service
    - namespaceSelector:
        matchLabels:
          name: cqpm  # Allow cqpm service
    ports:
    - protocol: TCP
      port: 9090
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
    - protocol: TCP
      port: 6379  # Redis
    - protocol: TCP
      port: 4222  # NATS
```

### Authentication

**API Key Management**:
- Generate strong API keys (32+ characters)
- Rotate keys quarterly
- Use separate keys per consuming service
- Store keys in secrets manager (AWS Secrets Manager, HashiCorp Vault)

**Generate API Key**:
```bash
# Generate random API key
openssl rand -base64 32

# Add to secrets
kubectl create secret generic cqar-api-keys \
  --from-literal=cqmd-key='GENERATED_KEY_1' \
  --from-literal=cqpm-key='GENERATED_KEY_2' \
  -n cqar
```

### TLS/SSL

**Database TLS**:
```yaml
database:
  ssl_mode: "require"  # or "verify-full" with CA cert
  ssl_cert: "/certs/client-cert.pem"
  ssl_key: "/certs/client-key.pem"
  ssl_root_cert: "/certs/ca-cert.pem"
```

**gRPC TLS** (if not using service mesh):
```yaml
server:
  tls_enabled: true
  tls_cert_file: "/certs/server-cert.pem"
  tls_key_file: "/certs/server-key.pem"
```

**Service Mesh** (Recommended):
- Use Istio/Linkerd for automatic mTLS between services
- Offload TLS termination to sidecar proxy

### RBAC

**Kubernetes ServiceAccount**:
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cqar
  namespace: cqar
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: cqar-role
  namespace: cqar
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cqar-rolebinding
  namespace: cqar
subjects:
- kind: ServiceAccount
  name: cqar
  namespace: cqar
roleRef:
  kind: Role
  name: cqar-role
  apiGroup: rbac.authorization.k8s.io
```

---

## Scaling

### Horizontal Scaling

**Horizontal Pod Autoscaler**:
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: cqar-hpa
  namespace: cqar
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: cqar
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300  # 5 min
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 30
```

**Scaling Considerations**:
- CQAR is stateless, scales horizontally easily
- Database connection pool per replica: 25 connections
- Total connections = replicas × 25 (ensure db max_connections sufficient)
- Redis pool per replica: 10 connections

---

### Vertical Scaling

Increase resources for existing pods if needed:

```yaml
resources:
  requests:
    cpu: 1000m      # Increase from 500m
    memory: 1Gi     # Increase from 512Mi
  limits:
    cpu: 4000m      # Increase from 2000m
    memory: 4Gi     # Increase from 2Gi
```

---

### Database Scaling

**Read Replicas** (Future):
- Route read-only queries to replicas
- Write queries to primary
- Reduces primary load for high read workloads

**Connection Pooling**:
- Use PgBouncer for connection pooling
- Reduces connection overhead

**Sharding** (Future, if needed):
- Shard by asset_id or venue_id
- Requires application-level sharding logic

---

## Disaster Recovery

### Backup Strategy

**Database Backups**:
- **Frequency**: Daily automated backups
- **Retention**: 30 days
- **Type**: Full backup + WAL archiving
- **Storage**: S3/GCS with versioning

**Backup Script**:
```bash
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="cqar_backup_${DATE}.sql.gz"

pg_dump -h postgres.example.com \
        -U cqar_prod_user \
        -d cqar_prod \
        | gzip > /backups/${BACKUP_FILE}

# Upload to S3
aws s3 cp /backups/${BACKUP_FILE} s3://cqar-backups/daily/

# Cleanup old backups (>30 days)
find /backups -name "cqar_backup_*.sql.gz" -mtime +30 -delete
```

**Automated Backup (CronJob)**:
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cqar-backup
  namespace: cqar
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:14
            command: ["/bin/sh"]
            args:
            - -c
            - |
              pg_dump $DATABASE_URL | gzip | aws s3 cp - s3://cqar-backups/daily/cqar_$(date +%Y%m%d).sql.gz
            env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: cqar-secrets
                  key: database-url
          restartPolicy: OnFailure
```

---

### Recovery Procedures

**Restore from Backup**:
```bash
# Download backup
aws s3 cp s3://cqar-backups/daily/cqar_20251016.sql.gz .

# Restore
gunzip cqar_20251016.sql.gz
psql -h postgres.example.com \
     -U cqar_prod_user \
     -d cqar_prod \
     < cqar_20251016.sql
```

**Point-in-Time Recovery** (PITR):
- Requires WAL archiving enabled
- Restore to specific timestamp between backups

---

### Multi-Region Failover

**Active-Passive**:
- Primary region: Active CQAR + database
- Secondary region: Standby CQAR + database replica
- Manual failover via DNS/load balancer change

**Active-Active** (Future):
- Multi-region write replication (complex)
- Requires conflict resolution strategy

---

**Related Documentation**:
- [OPERATIONS.md](OPERATIONS.md) - Operational procedures
- [API.md](API.md) - API reference
- [README.md](../README.md) - Getting started
