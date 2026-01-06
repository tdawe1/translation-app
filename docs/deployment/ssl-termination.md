# SSL/TLS Termination

Encryption in transit is non-negotiable for GengoWatcher SaaS. we use SSL/TLS to protect user credentials, session tokens, and job data.

## 1. Termination Points

Depending on your deployment strategy, SSL termination happens at different layers:

### A. Load Balancer (Cloud)
If using AWS ALB or DigitalOcean Load Balancer, upload your certificates to their respective certificate managers. The LB handles decryption and forwards plain HTTP/1.1 traffic to the app within your private network.

### B. Nginx Ingress (Kubernetes)
Using **Cert-Manager** with the **Let's Encrypt** issuer is the recommended approach for K8s. It handles automatic issuance and renewal of certificates.

### C. Standalone Nginx (Docker)
Manual certificate management using **Certbot**.

---

## 2. Setting Up Certbot (Let's Encrypt)

On a standalone Linux server:

1. **Install Certbot**:
   ```bash
   sudo apt install certbot python3-certbot-nginx
   ```
2. **Generate Certificates**:
   ```bash
   sudo certbot --nginx -d gengowatcher.com -d api.gengowatcher.com
   ```
3. **Automatic Renewal**: Certbot adds a cron job automatically. You can test it with:
   ```bash
   sudo certbot renew --dry-run
   ```

---

## 3. Strong SSL Configuration

We recommend using the **Intermediate** profile from the [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/).

- **Protocols**: TLSv1.2, TLSv1.3.
- **Ciphers**: Modern, secure ciphers avoiding those with known vulnerabilities.
- **HSTS**: (HTTP Strict Transport Security) Instructs browsers to only communicate via HTTPS for the next year.
  ```nginx
  add_header Strict-Transport-Security "max-age=63072000" always;
  ```

---

## 4. WebSocket Security (WSS)

When SSL is enabled, WebSockets **must** use the `wss://` protocol. Browsers will block `ws://` connections from an `https://` page (Mixed Content error).

The Nginx configuration must correctly pass the `Upgrade` and `Connection` headers to maintain the secure WebSocket tunnel.

## 5. Security Checklist
- [ ] No SSLv2 or SSLv3 enabled.
- [ ] No vulnerable ciphers (e.g., RC4, 3DES).
- [ ] Heartbleed vulnerability patched (OpenSSL updated).
- [ ] A+ rating on [SSL Labs Test](https://www.ssllabs.com/ssltest/).

## Next Steps
- [Nginx Configuration](../deployment/nginx-configuration.md)
- [Production Setup Checklist](../deployment/production-setup.md)
