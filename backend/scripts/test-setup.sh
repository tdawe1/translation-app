#!/bin/bash
# Test setup script - ensures test database is running

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  GengoWatcher Test Setup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if docker is available
if ! command -v docker &> /dev/null; then
    echo "Error: docker is not installed or not in PATH"
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "Error: docker-compose is not installed or not in PATH"
    exit 1
fi

# Use docker compose or docker-compose
COMPOSE_CMD="docker-compose"
if docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
fi

# Start test database
echo "Starting test database..."
$COMPOSE_CMD -f backend/docker-compose.test.yml up -d

# Wait for database to be ready
echo "Waiting for database to be ready..."
MAX_TRIES=30
COUNT=0
while [ $COUNT -lt $MAX_TRIES ]; do
    if docker exec gengowatcher-postgres-test pg_isready -U gengo &> /dev/null; then
        echo "Database is ready!"
        break
    fi
    COUNT=$((COUNT + 1))
    echo "Waiting... ($COUNT/$MAX_TRIES)"
    sleep 1
done

if [ $COUNT -eq $MAX_TRIES ]; then
    echo "Error: Database failed to become ready"
    $COMPOSE_CMD -f backend/docker-compose.test.yml logs
    exit 1
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test database ready!"
echo "  Database: gengowatcher_test"
echo "  Host:     localhost:5433"
echo "  User:     gengo"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Run tests with:"
echo "  cd backend && make test"
echo ""
echo "Stop database with:"
echo "  $COMPOSE_CMD -f backend/docker-compose.test.yml down"
