# Architecture Documentation

## System Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Load Balancer / CDN                             │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                    ┌──────────────────┼──────────────────┐
                    │                  │                  │
           ┌────────▼────────┐ ┌───────▼────────┐ ┌──────▼────────┐
           │   Nginx (SSL)   │ │   Frontend     │ │   Backend     │
           │   Rate Limiting │ │   Next.js 16   │ │   Go + Fiber  │
           └────────┬────────┘ └───────┬────────┘ └───────┬────────┘
                    │                  │                  │
                    │         ┌────────▼────────┐        │
                    │         │   PostgreSQL    │        │
                    │         │   (Primary DB)  │        │
                    │         └────────┬────────┘        │
                    │                  │                  │
                    │         ┌────────▼────────┐        │
                    │         │     Redis       │        │
                    │         │  (Cache/PubSub) │        │
                    │         └─────────────────┘        │
                    └────────────────────────────────────┘
```

## Component Details

### Frontend (Next.js 16)

```
frontend/
├── app/                    # Next.js App Router
│   ├── auth/              # Authentication pages
│   │   ├── login/         # Email/password login
│   │   ├── register/      # User registration
│   │   ├── magic-link/    # Magic link authentication
│   │   └── verify/        # Email verification
│   ├── dashboard/         # Main user dashboard
│   ├── settings/          # User settings
│   └── api/               # API routes (if needed)
├── components/            # React components
├── lib/                   # Utilities (API client, utils)
├── store/                 # Zustand state management
└── types/                 # TypeScript definitions
```

**Key Technologies:**
- React 19 with Server Components
- TypeScript for type safety
- Tailwind CSS for styling
- Zustand for state management
- TanStack Query for data fetching

### Backend (Go + Fiber)

```
backend/
├── cmd/server/            # Application entry point
├── internal/
│   ├── auth/              # Authentication logic
│   │   ├── service.go     # Auth business logic
│   │   ├── token.go       # JWT management
│   │   └── security.go    # Password hashing, encryption
│   ├── database/          # Database layer
│   │   ├── connection.go  # PostgreSQL connection
│   │   └── models.go      # GORM models
│   ├── handlers/          # HTTP handlers
│   │   ├── auth.go        # Auth endpoints
│   │   ├── oauth.go       # OAuth endpoints
│   │   ├── email.go       # Email verification
│   │   └── watcher.go     # Watcher management
│   ├── middleware/        # HTTP middleware
│   │   ├── jwt.go         # JWT validation
│   │   ├── ratelimit.go   # Rate limiting
│   │   └── csrf.go        # CSRF protection
│   ├── email/             # Email service (Resend)
│   ├── oauth/             # OAuth providers
│   ├── watcher/           # Job watching logic
│   ├── redis_pubsub.go    # Real-time notifications
│   └── config/            # Configuration management
├── alembic/               # Database migrations
└── tests/                 # Test files
```

**Key Technologies:**
- Fiber 3.x for HTTP routing
- GORM 2.0 for ORM
- golang-jwt/jwt for tokens
- go-redis for Redis
- bcrypt for password hashing

### Database Schema

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                 Users Table                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│ id (UUID)           │ PRIMARY KEY                                          │
│ email              │ UNIQUE, NOT NULL                                     │
│ email_verified     │ BOOLEAN DEFAULT FALSE                                │
│ password_hash      │ VARCHAR(255)                                         │
│ is_active          │ BOOLEAN DEFAULT TRUE                                 │
│ created_at         │ TIMESTAMP                                            │
│ updated_at         │ TIMESTAMP                                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
        ┌───────────▼───┐ ┌────────▼────────┐ ┌───▼─────────────┐
        │ OAuthAccounts │ │ MagicLinkTokens │ │ EmailVerifyTokens│
        ├───────────────┤ ├─────────────────┤ ├─────────────────┤
        │ user_id (FK)  │ │ email           │ │ email           │
        │ provider      │ │ token           │ │ token           │
        │ provider_user │ │ expires_at      │ │ expires_at      │
        └───────────────┘ └─────────────────┘ └─────────────────┘
```

## Data Flow

### Authentication Flow

```
1. User submits credentials
   ┌─────────┐      POST /api/v1/auth/login      ┌──────────┐
   │ Browser │ ───────────────────────────────▶ │ Backend  │
   └─────────┘                                   └────┬─────┘
                                                    │
                                           Validate credentials
                                                    │
                                           Generate JWT tokens
                                                    │
                                    ┌───────────────┴───────────────┐
                                    │                               │
                            Set httpOnly cookie              Return JSON
                                    │                               │
                            ┌──────▼──────┐                       │
                            │ Browser     │ ◀─────────────────────┘
                            │ Store cookie│
                            └──────┬──────┘
                                   │
                          Include cookie in requests
                                   │
                            ┌──────▼──────┐
                            │ Middleware  │ ◀── Validate JWT
                            │ JWTValidator│
                            └──────┬──────┘
                                   │
                            Add user_id to context
                                   │
                            ┌──────▼──────┐
                            │ Handler     │ ───▶ Business logic
                            └─────────────┘
```

### OAuth Flow

```
1. User clicks "Login with Google"
   ┌─────────┐      GET /api/v1/oauth/authorize?provider=google      ┌──────────┐
   │ Browser │ ───────────────────────────────────────────────────▶ │ Backend  │
   └─────────┘                                                    └────┬─────┘
                                                                  Generate state
                                                                  Store in memory
                                                                      │
                                                              Redirect to Google
                                                                      │
                                                              User authenticates with Google
                                                                      │
                                                              Google redirects with code
                                                                      │
                                    ┌───────────────────────────────┘
                                    │
                            Exchange code for token
                                    │
                            Fetch user info from Google
                                    │
                            Create/lookup user
                                    │
                            Generate session
                                    │
                            Set cookie + Return user
```

### Job Watching Flow

```
1. User configures watcher
   ┌─────────┐      PUT /api/v1/watcher/config         ┌──────────┐
   │ Browser │ ──────────────────────────────────────▶ │ Backend  │
   └─────────┘                                         └────┬─────┘
                                                          Store config in DB
                                                              │
2. User starts watcher                                    │
   ┌─────────┐      POST /api/v1/watcher/start    ┌───────▼───────┐
   │ Browser │ ───────────────────────────────▶ │ WatcherManager │
   └─────────┘                                  │ (Background)   │
                                                 └───────┬───────┘
                                                         │
                                          ┌──────────────┴──────────────┐
                                          │                             │
                                  Poll RSS feeds                WebSocket monitor
                                          │                             │
                                          │                      New job found
                                          │                             │
                                          │              ┌──────────────▼──────────┐
                                          │              │ Check against user criteria│
                                          │              └──────────────┬──────────┘
                                          │                             │
                                          │                    Match found
                                          │                             │
                                          │              ┌──────────────▼──────────┐
                                          │              │ Send via Redis pub/sub   │
                                          │              └──────────────┬──────────┘
                                          │                             │
                                          │              ┌──────────────▼──────────┐
                                          │              │ WebSocket sends to user  │
                                          │              └─────────────────────────┘
```

## Security Architecture

### Authentication Security

- **JWT Access Token**: 15-minute expiry
- **Refresh Token**: 7-day expiry (httpOnly cookie)
- **Password Hashing**: bcrypt with cost=12
- **OAuth State**: CSRF-protected with session binding

### API Security

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              API Request Pipeline                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. TLS Termination (Nginx)                                                  │
│     └── Ensures all traffic is encrypted                                     │
│                                                                              │
│  2. Rate Limiting (Nginx + Application)                                      │
│     ├── Auth endpoints: 3 req/min per IP                                    │
│     ├── Login: 10 req/min per IP                                            │
│     └── Email: 3 req/hour per IP                                            │
│                                                                              │
│  3. CORS Configuration                                                       │
│     ├── Whitelist allowed origins                                            │
│     ├── Allow credentials                                                    │
│     └── Specific headers                                                     │
│                                                                              │
│  4. JWT Validation                                                           │
│     ├── Verify signature                                                     │
│     ├── Check expiration                                                     │
│     └── Extract user claims                                                  │
│                                                                              │
│  5. Authorization                                                            │
│     ├── Verify user owns resource                                            │
│     └── Check subscription tier                                              │
│                                                                              │
│  6. Input Validation                                                         │
│     ├── Request body schema                                                  │
│     ├── Parameter validation                                                 │
│     └── SQL injection prevention (GORM)                                      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Isolation

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                             Multi-Tenant Isolation                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Each user has:                                                              │
│  ├── Dedicated watcher config record                                         │
│  ├── Dedicated watcher state record                                          │
│  ├── Redis keys: user:{user_id}:*                                           │
│  └── WebSocket room: user:{user_id}:ws                                      │
│                                                                              │
│  All database queries filter by user_id:                                     │
│  ```go                                                                       │
│  db.Where("user_id = ?", userID).First(&config)                             │
│  ```                                                                       │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Scalability Design

### Horizontal Scaling

```
                         ┌─────────────────┐
                         │   Load Balancer │
                         └────────┬────────┘
                                  │
              ┌───────────────────┼───────────────────┐
              │                   │                   │
      ┌───────▼───────┐   ┌───────▼───────┐   ┌───────▼───────┐
      │  Backend 1    │   │  Backend 2    │   │  Backend N    │
      │  (Go/Fiber)   │   │  (Go/Fiber)   │   │  (Go/Fiber)   │
      └───────┬───────┘   └───────┬───────┘   └───────┬───────┘
              │                   │                   │
              └───────────────────┼───────────────────┘
                                  │
              ┌───────────────────┼───────────────────┐
              │                   │                   │
      ┌───────▼───────┐   ┌───────▼───────┐   ┌───────▼───────┐
      │  PostgreSQL   │   │     Redis     │   │   (Optional)  │
      │  (Primary)    │   │  (Cluster)    │   │   Read Replica │
      └───────────────┘   └───────────────┘   └───────────────┘
```

### Caching Strategy

| Cache Type | Technology | TTL | Purpose |
|------------|------------|-----|---------|
| Session | Redis | 7 days | Refresh tokens |
| Rate Limit | Redis | Per request | Rate counting |
| Watcher State | Redis | Real-time | Live status |
| Static Assets | CDN | 1 year | Frontend assets |

## Monitoring & Observability

### Health Endpoints

- `GET /health` - Basic health check
- `GET /ready` - Readiness probe (includes DB check)

### Logging

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Logging Structure                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Format: JSON with structured fields                                         │
│  ```json                                                                     │
│  {                                                                           │
│    "level": "info",                                                          │
│    "timestamp": "2026-01-05T22:00:00Z",                                     │
│    "message": "User logged in",                                              │
│    "user_id": "uuid",                                                        │
│    "ip": "192.168.1.1",                                                      │
│    "request_id": "uuid"                                                      │
│  }                                                                           │
│  ```                                                                           │
│                                                                              │
│  Log Levels:                                                                 │
│  - ERROR: Request failures, exceptions                                       │
│  - WARN:  Recoverable errors, degraded performance                           │
│  - INFO:  Significant events (login, config change)                          │
│  - DEBUG: Detailed tracing for debugging                                     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Metrics (Ready for Integration)

| Metric | Type | Description |
|--------|------|-------------|
| http_requests_total | Counter | Total HTTP requests |
| http_request_duration_seconds | Histogram | Request latency |
| active_users | Gauge | Currently active users |
| watcher_instances | Gauge | Running watcher instances |
| jobs_found_total | Counter | Jobs discovered |
| websocket_connections | Gauge | Active WebSocket connections |

## Deployment Architecture

### Production Deployment

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Production Infrastructure                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                        Kubernetes Cluster                                │ │
│  │                                                                          │ │
│  │  ┌─────────────────────────────────────────────────────────────────┐    │ │
│  │  │                     Backend Deployment                          │    │ │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐                          │    │ │
│  │  │  │ Backend │ │ Backend │ │ Backend │ ... (HPA)               │    │ │
│  │  │  │ Pod 1   │ │ Pod 2   │ │ Pod 3   │                         │    │ │
│  │  │  └─────────┘ └─────────┘ └─────────┘                          │    │ │
│  │  └─────────────────────────────────────────────────────────────────┘    │ │
│  │                                                                          │ │
│  │  ┌─────────────────────────────────────────────────────────────────┐    │ │
│  │  │                     Frontend Deployment                         │    │ │
│  │  │  ┌─────────┐ ┌─────────┐                                        │    │ │
│  │  │  │ Next.js │ │ Next.js │ ... (HPA)                             │    │ │
│  │  │  │ Pod 1   │ │ Pod 2   │                                       │    │ │
│  │  │  └─────────┘ └─────────┘                                        │    │ │
│  │  └─────────────────────────────────────────────────────────────────┘    │ │
│  │                                                                          │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
│  External Services:                                                          │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │   PostgreSQL│ │    Redis    │ │   Resend    │ │    LemonSqueezy     │   │
│  │   (Cloud)   │ │   (Cloud)   │ │  (Email)    │ │    (Payments)       │   │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Error Handling

### Error Response Format

```json
{
  "error": "Descriptive error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "email",
    "reason": "invalid format"
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| INVALID_REQUEST | 400 | Malformed request |
| INVALID_CREDENTIALS | 401 | Wrong email/password |
| NOT_AUTHENTICATED | 401 | No valid token |
| USER_NOT_FOUND | 404 | Resource not found |
| RATE_LIMITED | 429 | Too many requests |
| INTERNAL_ERROR | 500 | Server-side error |

---

**Last Updated**: January 2026
**Version**: 1.0.0
