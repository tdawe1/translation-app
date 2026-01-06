# Environment Variables Reference

GengoWatcher SaaS is configured primarily through environment variables. This allows for clean separation of code and configuration across different environments (dev, staging, prod).

## 1. Core Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `ENV` | `development`, `staging`, or `production`. | `development` |
| `PORT` | The port the Go API server listens on. | `8000` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error`. | `info` |
| `JWT_SECRET` | Secret key for signing access tokens. | **REQUIRED** |

---

## 2. Database (PostgreSQL)

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | Database host address. | `localhost` |
| `DB_PORT` | Database port. | `5433` |
| `DB_USER` | Database username. | `gengo` |
| `DB_PASSWORD`| Database password. | `devpass` |
| `DB_NAME` | Database name. | `gengowatcher` |
| `DB_SSLMODE` | `disable`, `require`, `verify-full`. | `disable` |

---

## 3. Cache & Messaging (Redis)

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_URL` | Full connection string for Redis. | `redis://localhost:6379/0` |
| `REDIS_PASSWORD`| Redis password (if not in URL). | `""` |

---

## 4. Authentication & OAuth

| Variable | Description |
|----------|-------------|
| `GOOGLE_OAUTH_CLIENT_ID` | Client ID from Google Cloud Console. |
| `GOOGLE_OAUTH_CLIENT_SECRET` | Client Secret from Google Cloud Console. |
| `GITHUB_OAUTH_CLIENT_ID` | Client ID from GitHub Developer Settings. |
| `GITHUB_OAUTH_CLIENT_SECRET` | Client Secret from GitHub Developer Settings. |
| `OAUTH_REDIRECT_URL` | The URL users are redirected to after OAuth. |

---

## 5. Third-Party Services

| Variable | Description |
|----------|-------------|
| `RESEND_API_KEY` | API key from Resend.com. |
| `LEMONSQUEEZY_WEBHOOK_SECRET` | Secret for verifying billing webhooks. |
| `SUPPORT_EMAIL` | Contact email shown in footer/support. |

---

## 6. Frontend (Next.js)

| Variable | Description | Default |
|----------|-------------|---------|
| `NEXT_PUBLIC_API_URL`| The public URL of your Backend API. | `http://localhost:8000` |
| `NEXT_PUBLIC_WS_URL` | The public URL of your WebSocket server. | `ws://localhost:8000` |

## Best Practices
1. **Security**: Never commit `.env` files to git. Use `.env.example` as a template.
2. **Persistence**: In production, use a dedicated secret manager (e.g., AWS Secrets Manager).
3. **Naming**: Use uppercase with underscores for all environment variables.

## Next Steps
- [Database Configuration](../configuration/database-configuration.md)
- [Redis Configuration](../configuration/redis-configuration.md)
