# Deployment Documentation

## Overview

This guide covers deploying GengoWatcher to production. Choose your deployment method:

| Method | Description | Complexity |
|--------|-------------|------------|
| Docker Compose | Single-server deployment | Easy |
| Kubernetes | Scalable, cloud-native | Advanced |
| Manual | Custom infrastructure | Expert |

---

## Prerequisites

### Required Accounts

- [ ] PostgreSQL database (managed or self-hosted)
- [ ] Redis instance (managed or self-hosted)
- [ ] Resend account (for emails)
- [ ] Google Cloud project (for OAuth)
- [ ] GitHub OAuth App (for OAuth)
- [ ] LemonSqueezy account (for payments - optional)

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Docker | 24.x+ | Container runtime |
| Docker Compose | 2.x+ | Container orchestration |
| OpenSSL | Any | Secret generation |
| Git | Any | Deployment |

---

## Environment Configuration

### 1. Generate Secrets

```bash
# Generate JWT secret (minimum 32 characters)
JWT_SECRET=$(openssl rand -base64 32)
echo "JWT_SECRET=$JWT_SECRET"

# Generate webhook secret
WEBHOOK_SECRET=$(openssl rand -base64 24)
echo "LEMONSQUEEZY_WEBHOOK_SECRET=$WEBHOOK_SECRET"
```

### 2. Create Environment File

```bash
cd /home/thomas/translation-app/backend
cp .env.production.example .env.production
```

### 3. Configure Environment Variables

Edit `.env.production`:

```bash
# Server
ENV=production
PORT=8000

# Database
DB_HOST=your-db-host.rds.amazonaws.com
DB_PORT=5432
DB_USER=gengowatcher
DB_PASSWORD=your-secure-password
DB_NAME=gengowatcher_production
DB_SSLMODE=require

# Redis
REDIS_URL=redis://:password@your-redis-host:6379/0

# JWT (use generated value)
JWT_SECRET=your-generated-secret

# Email
RESEND_API_KEY=re_xxxxxxxxxxxx
FROM_EMAIL=noreply@yourdomain.com
FROM_NAME=GengoWatcher

# CORS
ALLOWED_ORIGINS=https://app.yourdomain.com
OAUTH_REDIRECT_URL=https://api.yourdomain.com

# OAuth (from Google Cloud Console)
GOOGLE_OAUTH_CLIENT_ID=xxx.apps.googleusercontent.com
GOOGLE_OAUTH_CLIENT_SECRET=xxx

# OAuth (from GitHub Settings)
GITHUB_OAUTH_CLIENT_ID=xxx
GITHUB_OAUTH_CLIENT_SECRET=xxx

# Payments (from LemonSqueezy)
LEMONSQUEEZY_WEBHOOK_SECRET=whsec_xxx
```

---

## Docker Compose Deployment (Recommended)

### 1. Create Production Compose File

```bash
cd /home/thomas/translation-app
cp docker-compose.production.yml docker-compose.yml
```

### 2. Configure Nginx (Optional)

If using included nginx:

```bash
mkdir -p deploy/nginx/ssl
# Copy your SSL certificates
cp your-domain.crt deploy/nginx/ssl/fullchain.pem
cp your-domain.key deploy/nginx/ssl/privkey.pem
```

### 3. Build and Start

```bash
# Build images
docker-compose build --no-cache

# Start services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f backend
```

### 4. Verify Deployment

```bash
# Health check
curl https://api.yourdomain.com/health

# Expected response:
{"status":"healthy","service":"gengowatcher-saas"}
```

### 5. Stop Deployment

```bash
docker-compose down
```

---

## Kubernetes Deployment

### 1. Create Namespace

```bash
kubectl apply -f deploy/k8s/namespace.yaml
```

### 2. Create Secrets

```bash
# Create secret from environment file
kubectl create secret generic gengowatcher-secrets \
  --from-env-file=.env.production \
  --namespace=gengowatcher
```

### 3. Deploy Backend

```bash
kubectl apply -f deploy/k8s/backend-deployment.yaml
```

### 4. Verify Deployment

```bash
# Check pods
kubectl get pods -n gengowatcher

# Check logs
kubectl logs -l app=gengowatcher -n gengowatcher -f

# Check service
kubectl get svc -n gengowatcher
```

### 5. Configure Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gengowatcher-ingress
  namespace: gengowatcher
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - api.yourdomain.com
        - app.yourdomain.com
      secretName: gengowatcher-tls
  rules:
    - host: api.yourdomain.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: gengowatcher-backend
                port:
                  number: 8000
    - host: app.yourdomain.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: gengowatcher-frontend
                port:
                  number: 3000
```

```bash
kubectl apply -f ingress.yaml
```

---

## Database Setup

### 1. Run Migrations

```bash
# Using Docker
docker-compose exec backend alembic upgrade head

# Or directly
cd backend
pip install alembic
alembic upgrade head
```

### 2. Verify Tables

```bash
psql -h $DB_HOST -U $DB_USER $DB_NAME -c "\dt"
```

Expected tables:
- users
- oauth_accounts
- magic_link_tokens
- email_verification_tokens
- password_reset_tokens
- refresh_tokens
- watcher_configs
- watcher_states

---

## OAuth Configuration

### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Create OAuth 2.0 Client ID
3. Configure:
   - **Authorized redirect URIs:**
     ```
     https://api.yourdomain.com/api/v1/oauth/google/callback
     ```
4. Copy Client ID and Secret to `.env.production`

### GitHub OAuth

1. Go to [GitHub OAuth Apps](https://github.com/settings/developers)
2. Create new OAuth Application
3. Configure:
   - **Authorization callback URL:**
     ```
     https://api.yourdomain.com/api/v1/oauth/github/callback
     ```
4. Copy Client ID and Secret to `.env.production`

---

## Email Configuration (Resend)

1. Sign up at [Resend.com](https://resend.com)
2. Add domain and verify DNS records
3. Generate API key
4. Add to `.env.production`:
   ```
   RESEND_API_KEY=re_xxxxxxxxxxxx
   FROM_EMAIL=noreply@yourdomain.com
   ```

---

## Production Checklist

### Pre-Deployment

- [ ] All secrets generated and stored securely
- [ ] Database created and accessible
- [ ] Redis instance running
- [ ] SSL certificates obtained
- [ ] Domain DNS configured
- [ ] OAuth apps configured with correct callbacks
- [ ] Email domain verified (Resend)

### Deployment

- [ ] Code deployed to server/repository
- [ ] Docker images built
- [ ] Migrations applied
- [ ] Services running
- [ ] Health check passing
- [ ] Logs accessible

### Post-Deployment

- [ ] Test user registration
- [ ] Test OAuth login (Google)
- [ ] Test OAuth login (GitHub)
- [ ] Test magic link
- [ ] Test job watcher configuration
- [ ] Test WebSocket connection
- [ ] Verify rate limiting
- [ ] Check error logging

---

## Rollback Procedure

### Docker Compose Rollback

```bash
# Stop current deployment
docker-compose down

# Restore database from backup
gunzip backups/db_backup_YYYYMMDD_HHMMSS.sql.gz | psql -h $DB_HOST -U $DB_USER $DB_NAME

# Start previous version (if using tags)
docker-compose pull
docker-compose up -d
```

### Kubernetes Rollback

```bash
# Rollback deployment
kubectl rollout undo deployment/gengowatcher-backend -n gengowatcher

# If needed, restore database
kubectl exec -it backup-job-xxx -n gengowatcher -- pg_restore ...
```

---

## Monitoring

### Health Check Endpoint

```bash
curl https://api.yourdomain.com/health
```

### Logs

```bash
# Docker Compose
docker-compose logs -f backend

# Kubernetes
kubectl logs -l app=gengowatcher -n gengowatcher -f
```

### Metrics (Ready for Prometheus Integration)

| Metric | Description |
|--------|-------------|
| `http_requests_total` | Total HTTP requests |
| `http_request_duration_seconds` | Request latency |
| `active_users` | Currently active users |
| `watcher_instances` | Running watchers |
| `jobs_found_total` | Jobs discovered |

---

## Troubleshooting

### Services Not Starting

```bash
# Check logs
docker-compose logs backend

# Common issues:
# - Database connection refused (check DB_HOST, DB_PORT)
# - Redis connection refused (check REDIS_URL)
# - Missing environment variables
```

### Database Connection Failed

```bash
# Test connection
psql -h $DB_HOST -p $DB_PORT -U $DB_USER $DB_NAME

# Check if database exists
psql -h $DB_HOST -U $DB_USER -l
```

### OAuth Not Working

1. Verify redirect URIs in OAuth provider settings
2. Check Client ID/Secret in environment
3. Ensure CORS allows your frontend domain

### Emails Not Sending

1. Verify Resend API key
2. Check domain is verified in Resend
3. Look for errors in logs:
   ```bash
   docker-compose logs backend | grep -i email
   ```

---

## Scaling

### Horizontal Scaling (Kubernetes)

The backend deployment includes HPA configuration:

```yaml
# From deploy/k8s/backend-deployment.yaml
autoscaling/v2:
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

### Vertical Scaling

Increase resource limits in deployment:

```yaml
resources:
  requests:
    cpu: "200m"
    memory: "256Mi"
  limits:
    cpu: "1000m"
    memory: "1Gi"
```

---

## Security Hardening

### 1. Enable Audit Logging

Configure audit logs for sensitive operations.

### 2. Set Up WAF

Use Cloudflare or AWS WAF for:
- SQL injection protection
- Cross-site scripting (XSS) protection
- Rate limiting at edge

### 3. Regular Updates

```bash
# Update dependencies
cd backend && go get -u ./...
cd frontend && npm update

# Rebuild and redeploy
docker-compose build --no-cache
```

### 4. Secret Rotation

Rotate secrets regularly:

```bash
# Generate new JWT secret
JWT_SECRET=$(openssl rand -base64 32)

# Update environment and redeploy
# Force all users to re-login
```

---

**Last Updated**: January 2026
**Version**: 1.0.0
