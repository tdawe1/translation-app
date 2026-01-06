# Security Best Practices

Security is paramount for a multi-tenant SaaS. GengoWatcher follows industry-standard security practices to protect user data and ensure system integrity.

## 1. Authentication & Access Control

- **JWT Rotation**: Access tokens expire in 15 minutes. Refresh tokens are rotated on every use.
- **Password Hashing**: We use **Argon2id** with high-cost parameters.
- **Social Login**: OAuth 2.0 implementation uses the `state` parameter to prevent CSRF.
- **API Keys**: Stored as salted hashes in the database; users can only see the full key once upon creation.

---

## 2. Infrastructure Security

- **Encryption in Transit**: All traffic is enforced over TLS (HTTPS/WSS).
- **Encryption at Rest**: PostgreSQL volumes should be encrypted using cloud provider tools (e.g., AWS EBS encryption).
- **Network Isolation**: PostgreSQL and Redis should not be accessible from the public internet. Only the Backend API should have access.

---

## 3. Application Security

- **Rate Limiting**: Applied to all endpoints to prevent DDoS and brute-force attacks.
- **SQL Injection**: We use **GORM** for all database queries, which uses prepared statements by default.
- **Input Validation**: All request bodies are strictly validated against JSON schemas.
- **Output Sanitization**: We ensure that no sensitive user data (like password hashes or internal IDs) is leaked in API responses.

---

## 4. Multi-Tenant Protection

- **Scoped Queries**: Every database query is scoped by `user_id`.
- **WebSocket Isolation**: Users are subscribed to unique Redis channels based on their `user_id`.
- **Quota Enforcement**: Subscription tiers are checked before intensive operations (like starting new watchers).

---

## 5. Maintenance & Compliance

- **Dependency Scanning**: Regularly run `go list -m all` and `pnpm audit` to check for vulnerable packages.
- **Secret Management**: Never commit secrets to version control. Use environment variables or secret managers (e.g., AWS Secrets Manager, HashiCorp Vault).
- **Audit Logs**: Maintain a permanent log of all security-sensitive actions (logins, permission changes, administrative overrides).

## Next Steps
- [Authentication Guide](../getting-started/authentication.md)
- [Production Setup](../deployment/production-setup.md)
