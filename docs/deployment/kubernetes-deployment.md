# Kubernetes Deployment Guide

For high-availability, GengoWatcher SaaS is designed to be deployed on a Kubernetes cluster. We provide manifests for Deployments, Services, and Horizontal Pod Autoscalers.

## 1. Prerequisites
- A running Kubernetes cluster (v1.28+ recommended).
- `kubectl` configured on your local machine.
- A private container registry (e.g., AWS ECR, Google GCR).
- **Cert-Manager** installed for SSL certificate management.

---

## 2. Secrets Management

Do not store secrets in your manifests. Create a `Secret` resource in your namespace:

```bash
kubectl create secret generic gengowatcher-secrets \
  --from-literal=JWT_SECRET='your_secret' \
  --from-literal=DATABASE_URL='postgres://...' \
  --from-literal=RESEND_API_KEY='...'
```

---

## 3. Deployment Manifests

### Backend Deployment
Our backend is stateless and supports horizontal scaling.
- **Replicas**: Minimum 2 for HA.
- **Probes**: Includes `livenessProbe` and `readinessProbe` targeting `/health` and `/ready`.

### Frontend Deployment
The Next.js frontend is deployed as a separate service, optimized for serving static assets and server-side rendering.

---

## 4. Ingress Configuration

We use an **Ingress Controller** (e.g., Nginx Ingress) to handle incoming traffic and SSL termination.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gengowatcher-ingress
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - api.gengowatcher.com
    - app.gengowatcher.com
    secretName: gengowatcher-tls
  rules:
  - host: api.gengowatcher.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: backend-service
            port:
              number: 8000
```

---

## 5. Autoscaling (HPA)

The backend scales automatically based on CPU utilization.

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: backend-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: backend-deployment
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

---

## 6. Updates & Rollbacks

Deploy a new version by updating the image tag:
```bash
kubectl set image deployment/backend-deployment backend=your-repo/backend:v2.0
```

Monitor the rollout:
```bash
kubectl rollout status deployment/backend-deployment
```

If issues are detected, rollback immediately:
```bash
kubectl rollout undo deployment/backend-deployment
```

## Next Steps
- [Scaling Strategies](../deployment/scaling-strategies.md)
- [Monitoring & Alerts](../deployment/monitoring-alerts.md)
