# Nginx Configuration

Nginx acts as our reverse proxy, load balancer, and SSL termination point. In production, it sits in front of both the Backend API and the Next.js Frontend.

## 1. Role of Nginx
- **Routing**: Directs traffic to the correct service (e.g., `/api` to Backend, `/` to Frontend).
- **SSL/TLS**: Handles encryption and decryption.
- **Compression**: Gzip/Brotli compression for faster asset delivery.
- **Rate Limiting**: Provides an initial layer of defense against DDoS attacks.
- **WebSocket Support**: Handles the `Upgrade` header for long-lived watcher connections.

---

## 2. Configuration Example

```nginx
upstream backend_api {
    server backend:8000;
}

upstream frontend_app {
    server frontend:3000;
}

server {
    listen 443 ssl http2;
    server_name gengowatcher.com;

    # SSL Certificates
    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;

    # Gzip Compression
    gzip on;
    gzip_types text/plain text/css application/json application/javascript;

    # Frontend
    location / {
        proxy_pass http://frontend_app;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # Backend API
    location /api/v1/ {
        proxy_pass http://backend_api;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # WebSockets
    location /ws {
        proxy_pass http://backend_api;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400s; # 24 hours
    }
}
```

---

## 3. Rate Limiting at the Edge

You can add a basic rate limit in the `http` block:

```nginx
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;

server {
    location /api/v1/ {
        limit_req zone=api_limit burst=20 nodelay;
        proxy_pass http://backend_api;
    }
}
```

---

## 4. Security Headers

We include security-enhancing headers in all Nginx responses:
```nginx
add_header X-Frame-Options "SAMEORIGIN";
add_header X-XSS-Protection "1; mode=block";
add_header X-Content-Type-Options "nosniff";
add_header Referrer-Policy "no-referrer-when-downgrade";
add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'";
```

---

## 5. Deployment
When using Docker, Nginx is deployed as its own container. Ensure the `nginx.conf` is mounted as a volume.

```bash
docker run -d --name nginx -p 80:80 -p 443:443 -v ./nginx.conf:/etc/nginx/nginx.conf nginx
```

## Next Steps
- [SSL Termination](../deployment/ssl-termination.md)
- [Docker Deployment Guide](../deployment/docker-deployment.md)
