# Kubernetes Deployment

Three deployment methods are provided. Pick whichever fits your workflow.

## Quick Start

### Plain Manifests

```sh
kubectl apply -f manifests/
```

### Helm

```sh
helm install gospeed helm/gospeed/

# With custom values
helm install gospeed helm/gospeed/ \
  --set image.repository=ghcr.io/goozt/gospeed-server \
  --set image.tag=v1.3.2 \
  --set ingress.enabled=true \
  --set autoscaling.enabled=true
```

### Kustomize

```sh
# Dev (single replica, no extras)
kubectl apply -k kustomize/overlays/dev/

# Production (HPA, ingress, network policy)
kubectl apply -k kustomize/overlays/prod/
```

## Architecture

```
k8s/
├── manifests/           # Standalone YAML — apply directly
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── deployment.yaml
│   ├── service.yaml     # Separate TCP + UDP services
│   ├── ingress.yaml
│   ├── hpa.yaml
│   ├── networkpolicy.yaml
│   └── pvc.yaml         # ACME cert persistence
│
├── helm/gospeed/        # Helm chart with values.yaml
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│
└── kustomize/
    ├── base/            # Minimal deployment
    └── overlays/
        ├── dev/         # 1 replica, no TLS/HPA/ingress
        └── prod/        # 2+ replicas, full stack
```

## Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 9000 | TCP | Control channel + TCP data streams |
| 9000 | UDP | UDP throughput, jitter, MTU tests |
| 8080 | TCP | Health check (`GET /health`) |

TCP and UDP require separate Kubernetes Services since a single Service cannot mix protocols.

## Configuration

### TLS

**Self-signed** (quickest for testing):
```sh
helm install gospeed helm/gospeed/ --set tls.enabled=true --set tls.mode=self-signed
```

**ACME / Let's Encrypt** (needs persistence for cert cache):
```sh
helm install gospeed helm/gospeed/ \
  --set tls.enabled=true \
  --set tls.mode=acme \
  --set tls.domain=speed.example.com \
  --set tls.email=admin@example.com \
  --set persistence.enabled=true
```

### Autoscaling

```sh
helm install gospeed helm/gospeed/ \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=2 \
  --set autoscaling.maxReplicas=10
```

### Network Policy

```sh
helm install gospeed helm/gospeed/ --set networkPolicy.enabled=true
```

## Health Checks

The deployment uses the built-in `--health` flag which starts an HTTP server on port 8080:

- **Liveness probe**: `GET /health` — restarts the pod if unresponsive
- **Readiness probe**: `GET /health` — removes pod from service endpoints until ready

## Security

- Runs as non-root user (UID 65534) on distroless base image
- Read-only root filesystem
- All Linux capabilities dropped
- No privilege escalation allowed
