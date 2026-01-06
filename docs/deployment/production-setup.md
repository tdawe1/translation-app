# Production Deployment Setup

This guide provides a comprehensive checklist and high-level overview for deploying GengoWatcher SaaS to a production environment.

## 1. Infrastructure Requirements

Before deploying, ensure you have the following resources provisioned:

- **Compute**: A server (e.g., AWS EC2, DigitalOcean Droplet) or a Kubernetes cluster.
- **Database**: A managed PostgreSQL instance (e.g., AWS RDS, Supabase).
- **Cache**: A managed Redis instance (e.g., AWS ElastiCache, Upstash).
- **Domain**: A registered domain name with access to DNS records.
- **SSL**: Certificates for your domain (e.g., via Let's Encrypt).

---

## 2. Security Checklist

- [ ] **Secrets**: All secrets (`JWT_SECRET`, API keys) are generated securely and NOT committed to Git.
- [ ] **Firewall**: Only ports 80 and 443 are open to the public. PostgreSQL and Redis are restricted to internal traffic.
- [ ] **SSH**: Root login is disabled, and SSH key authentication is enforced.
- [ ] **Updates**: The server OS and Docker runtime are up to date.

---

## 3. Pre-Deployment Steps

1. **Build Docker Images**:
   ```bash
   docker build -t gengowatcher-backend:latest ./backend
   docker build -t gengowatcher-frontend:latest ./frontend
   ```
2. **Push to Registry**: Push your images to a private registry (e.g., Docker Hub, AWS ECR).
3. **Configure Environment**: Create your production `.env` files or populate your secret manager.

---

## 4. Database Migration

Run migrations against the production database before launching the application:
```bash
cd backend
export DATABASE_URL=your_production_url
alembic upgrade head
```

---

## 5. Deployment Methods

We support two primary production deployment paths:

### A. Docker Compose (Single Server)
Ideal for small to medium installations.
- See: [Docker Deployment Guide](../deployment/docker-deployment.md)

### B. Kubernetes (Scalable Cluster)
Recommended for high-availability and large-scale deployments.
- See: [Kubernetes Deployment Guide](../deployment/kubernetes-deployment.md)

---

## 6. Post-Deployment Verification

1. **Health Check**: `curl https://api.yourdomain.com/health`
2. **Dashboard**: Log in and verify that you can navigate through the app.
3. **Watcher**: Start a test watcher and verify it discovers jobs.
4. **Logs**: Check your logging service for any startup errors.

## Next Steps
- [Docker Deployment Guide](../deployment/docker-deployment.md)
- [Kubernetes Deployment Guide](../deployment/kubernetes-deployment.md)
- [Nginx Configuration](../deployment/nginx-configuration.md)
