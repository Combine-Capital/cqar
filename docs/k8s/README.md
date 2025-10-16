# Kubernetes Manifests for CQAR

This directory contains Kubernetes manifests for deploying CQAR in production.

## Files

- **namespace.yaml**: Creates `cqar` namespace
- **configmap.yaml**: Non-sensitive configuration
- **secret.yaml.example**: Template for sensitive data (DO NOT commit real secrets!)
- **rbac.yaml**: ServiceAccount, Role, RoleBinding for RBAC
- **deployment.yaml**: Main CQAR deployment with 3 replicas
- **service.yaml**: ClusterIP services for gRPC, HTTP, metrics
- **ingress.yaml**: Load balancer / ingress configuration
- **hpa.yaml**: Horizontal Pod Autoscaler (3-10 replicas)
- **network-policy.yaml**: Network isolation rules
- **pvc.yaml**: Persistent volume for logs (optional)

## Quick Deployment

### 1. Create Namespace
```bash
kubectl apply -f namespace.yaml
```

### 2. Create Secrets
**Option A: Manual (dev/staging)**
```bash
kubectl create secret generic cqar-secrets \
  --from-literal=database-password='YOUR_DB_PASSWORD' \
  --from-literal=redis-password='YOUR_REDIS_PASSWORD' \
  --from-literal=api-keys='key1:service1,key2:service2' \
  -n cqar
```

**Option B: External Secrets Operator (production)**
```bash
# Install External Secrets Operator first
# Then configure SecretStore and apply external-secret.yaml
kubectl apply -f external-secret.yaml
```

### 3. Apply ConfigMap
```bash
kubectl apply -f configmap.yaml
```

### 4. Apply RBAC
```bash
kubectl apply -f rbac.yaml
```

### 5. Deploy Application
```bash
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

### 6. Configure Autoscaling
```bash
kubectl apply -f hpa.yaml
```

### 7. Apply Network Policy (optional but recommended)
```bash
kubectl apply -f network-policy.yaml
```

### 8. Expose via Ingress
```bash
kubectl apply -f ingress.yaml
```

## Verify Deployment

```bash
# Check pods
kubectl get pods -n cqar

# Check services
kubectl get svc -n cqar

# Check logs
kubectl logs -n cqar -l app=cqar --tail=50

# Check health
kubectl port-forward -n cqar svc/cqar-http 8080:8080
curl http://localhost:8080/health/ready
```

## Rollout Updates

```bash
# Update image
kubectl set image deployment/cqar cqar=gcr.io/combine-capital/cqar:0.2.0 -n cqar

# Check rollout status
kubectl rollout status deployment/cqar -n cqar

# Rollback if needed
kubectl rollout undo deployment/cqar -n cqar
```

## Scaling

```bash
# Manual scaling
kubectl scale deployment/cqar --replicas=5 -n cqar

# HPA will automatically scale based on CPU/memory
# Check HPA status
kubectl get hpa -n cqar
```

## Troubleshooting

```bash
# Describe pod
kubectl describe pod -n cqar <pod-name>

# Get pod logs
kubectl logs -n cqar <pod-name> --tail=100 -f

# Exec into pod
kubectl exec -it -n cqar <pod-name> -- /bin/sh

# Check events
kubectl get events -n cqar --sort-by='.lastTimestamp'
```

## Production Checklist

- [ ] Secrets stored in secrets manager (not in git)
- [ ] Resource limits configured (CPU, memory)
- [ ] Health checks configured (liveness, readiness)
- [ ] HPA configured for autoscaling
- [ ] Network policies applied
- [ ] Anti-affinity rules spread pods across nodes/zones
- [ ] Ingress/load balancer configured with TLS
- [ ] Monitoring configured (Prometheus scraping)
- [ ] Logging configured (Fluent Bit → Elasticsearch)
- [ ] Alerts configured (see docs/monitoring/)
- [ ] Backup strategy configured for database
- [ ] Disaster recovery plan documented

## Security Notes

1. **Never commit secrets to git** - Use secret.yaml.example as template only
2. **Use RBAC** - Principle of least privilege
3. **Enable network policies** - Restrict traffic to known services
4. **Run as non-root** - securityContext configured in deployment
5. **Use TLS** - For ingress and database connections
6. **Rotate credentials** - Quarterly API key rotation
7. **Scan images** - Use vulnerability scanners (Trivy, Snyk)

## Related Documentation

- [DEPLOYMENT.md](../DEPLOYMENT.md) - Complete deployment guide
- [OPERATIONS.md](../OPERATIONS.md) - Operational procedures
- [API.md](../API.md) - API reference
