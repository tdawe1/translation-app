# Docker Compose Deployment

For single-server production environments, GengoWatcher SaaS can be easily deployed using Docker Compose.

## 1. Directory Structure

On your production server, organize your files as follows:
```text
/opt/gengowatcher/
├── docker-compose.yml
├── .env.production
├── nginx/
│   └── nginx.conf
└── ssl/
    ├── cert.pem
    └── key.pem
```

---

## 2. The Production `docker-compose.yml`

This configuration focuses on stability and security, using pre-built images.

```yaml
version: '3.8'

services:
  backend:
    image: your-registry/gengowatcher-backend:latest
    env_file: .env.production
    restart: always
    depends_on:
      - postgres
      - redis

  frontend:
    image: your-registry/gengowatcher-frontend:latest
    restart: always
    ports:
      - "3000:3000"

  nginx:
    image: nginx:stable-alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - backend
      - frontend

  # Note: In production, it is recommended to use MANAGED 
  # Postgres/Redis instead of containers.
```

---

## 3. Deployment Workflow

1. **SSH into Server**:
   ```bash
   ssh user@your-server-ip
   ```
2. **Pull Latest Configuration**:
   ```bash
   cd /opt/gengowatcher
   git pull origin main
   ```
3. **Pull Images**:
   ```bash
   docker-compose pull
   ```
4. **Restart Services**:
   ```bash
   docker-compose up -d
   ```
5. **Clean Up**:
   ```bash
   docker image prune -f
   ```

---

## 4. Volume Persistence

Ensure your database data is persisted even if containers are removed:
```yaml
volumes:
  postgres_data:
```
*Note: If using managed databases (RDS/Cloud SQL), this is not necessary.*

---

## 5. Monitoring Logs

View real-time logs for all services:
```bash
docker-compose logs -f
```

To see logs for just the backend:
```bash
docker-compose logs -f backend
```

## Next Steps
- [Nginx Configuration](../deployment/nginx-configuration.md)
- [SSL Termination](../deployment/ssl-termination.md)
- [Production Setup Checklist](../deployment/production-setup.md)
