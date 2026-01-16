#!/bin/bash
# Smoke test runner with service health checks
# This script verifies all services are healthy before running Playwright tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Service configuration
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
BACKEND_URL="${BACKEND_URL:-http://localhost:8000}"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3001}"

echo "🔍 Checking service health..."

# 1. Check PostgreSQL
echo -n "  PostgreSQL (port $POSTGRES_PORT)... "
if pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" >/dev/null 2>&1; then
	echo -e "${GREEN}✓${NC}"
else
	echo -e "${RED}✗${NC}"
	echo -e "${RED}❌ PostgreSQL not ready${NC}"
	echo "💡 Run: ./scripts/dev.sh docker start"
	exit 1
fi

# 2. Check Redis
echo -n "  Redis (port $REDIS_PORT)... "
if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1; then
	echo -e "${GREEN}✓${NC}"
else
	echo -e "${RED}✗${NC}"
	echo -e "${RED}❌ Redis not ready${NC}"
	echo "💡 Run: ./scripts/dev.sh docker start"
	exit 1
fi

# 3. Check Backend (health endpoint)
echo -n "  Backend (${BACKEND_URL})... "
if curl -sf "${BACKEND_URL}/health" >/dev/null 2>&1; then
	echo -e "${GREEN}✓${NC}"
else
	echo -e "${RED}✗${NC}"
	echo -e "${RED}❌ Backend not ready${NC}"
	echo "💡 Run: ./scripts/dev.sh backend start"
	exit 1
fi

# 4. Check Frontend (responds on configured port)
echo -n "  Frontend (${FRONTEND_URL})... "
if curl -sf "$FRONTEND_URL" >/dev/null 2>&1; then
	echo -e "${GREEN}✓${NC}"
else
	echo -e "${RED}✗${NC}"
	echo -e "${RED}❌ Frontend not ready${NC}"
	echo "💡 Run: ./scripts/dev.sh frontend start"
	exit 1
fi

echo ""
echo -e "${GREEN}✅ All services healthy${NC}"
echo ""
echo "🧪 Running smoke tests..."
echo ""

# Run smoke tests
cd frontend
npm run test:smoke "$@"
