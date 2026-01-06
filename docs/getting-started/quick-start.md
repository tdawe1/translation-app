# Quick Start Guide

Get started with GengoWatcher SaaS in less than 5 minutes. This guide covers the essential steps for developers to set up their environment and for users to start monitoring jobs.

## For Users

### 1. Create an Account
Navigate to [GengoWatcher.com](https://gengowatcher.com) and sign up using one of our supported methods:
- **Email/Password**: Traditional sign-up.
- **OAuth**: One-click registration via Google or GitHub.
- **Magic Link**: Enter your email and click the link sent to your inbox.

### 2. Configure Your Watcher
Go to your **Dashboard** and set your monitoring criteria:
- **Language Pairs**: Select the languages you translate (e.g., English to Japanese).
- **Minimum Reward**: Set the lowest price point you're willing to accept.
- **RSS Feed URL**: Add your personal RSS feed from the translation platform.

### 3. Start Monitoring
Click the **"Start Watcher"** button. Your dashboard will now show real-time updates as jobs are discovered.

---

## For Developers (Local Setup)

Follow these steps to set up the GengoWatcher SaaS application on your local machine.

### Prerequisites
Ensure you have the following installed:
- **Go 1.23+**
- **Node.js 22+** (with `pnpm`)
- **Docker & Docker Compose**

### 1. Clone the Repository
```bash
git clone https://github.com/your-org/translation-app.git
cd translation-app
```

### 2. Start Infrastructure
Start the required services (PostgreSQL, Redis, MailHog) using Docker:
```bash
docker-compose up -d
```

### 3. Backend Setup
Configure and start the Go API server:
```bash
cd backend
cp .env.example .env
# Edit .env and set your JWT_SECRET
go run ./cmd/server
```
The API will be available at `http://localhost:8000`.

### 4. Frontend Setup
In a new terminal, set up the Next.js application:
```bash
cd frontend
pnpm install
pnpm dev
```
The dashboard will be available at `http://localhost:3000`.

### 5. Verify the API
Check if the backend is running correctly:
```bash
curl http://localhost:8000/health
# Expected: {"status":"healthy","service":"gengowatcher-saas"}
```

## Next Steps

- [Detailed Installation Guide](../getting-started/installation.md)
- [API Overview](../api/overview.md)
- [Authentication Details](../getting-started/authentication.md)
