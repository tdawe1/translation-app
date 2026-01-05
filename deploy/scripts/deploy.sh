#!/bin/bash
# =============================================================================
# GengoWatcher SaaS - Production Deployment Script
# =============================================================================
# Usage: ./deploy.sh [--build] [--migrate] [--rollback]
# =============================================================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="docker-compose.production.yml"
ENV_FILE=".env.production"
BACKUP_DIR="./backups"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    
    # Check env file
    if [ ! -f "$ENV_FILE" ]; then
        log_error "$ENV_FILE not found. Copy .env.production.example and fill in values."
        exit 1
    fi
    
    log_info "Prerequisites OK"
}

validate_env() {
    log_info "Validating environment variables..."
    
    source "$ENV_FILE"
    
    required_vars=(
        "JWT_SECRET"
        "DB_HOST"
        "DB_PASSWORD"
        "REDIS_URL"
    )
    
    for var in "${required_vars[@]}"; do
        if [ -z "${!var}" ]; then
            log_error "Required variable $var is not set in $ENV_FILE"
            exit 1
        fi
    done
    
    # Validate JWT_SECRET length
    if [ ${#JWT_SECRET} -lt 32 ]; then
        log_error "JWT_SECRET must be at least 32 characters"
        exit 1
    fi
    
    log_info "Environment validation OK"
}

backup_database() {
    log_info "Creating database backup..."
    
    mkdir -p "$BACKUP_DIR"
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_FILE="$BACKUP_DIR/db_backup_$TIMESTAMP.sql.gz"
    
    # This assumes you have pg_dump available or run it in a container
    docker-compose -f "$COMPOSE_FILE" exec -T backend pg_dump -U "$DB_USER" -h "$DB_HOST" "$DB_NAME" | gzip > "$BACKUP_FILE"
    
    log_info "Backup created: $BACKUP_FILE"
}

run_migrations() {
    log_info "Running database migrations..."
    
    # Run alembic migrations
    docker-compose -f "$COMPOSE_FILE" exec -T backend alembic upgrade head
    
    log_info "Migrations completed"
}

build_images() {
    log_info "Building Docker images..."
    
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" build --no-cache
    
    log_info "Build completed"
}

deploy() {
    log_info "Deploying services..."
    
    # Pull latest images (if using registry)
    # docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" pull
    
    # Start services with zero-downtime
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --remove-orphans
    
    log_info "Deployment completed"
}

health_check() {
    log_info "Running health checks..."
    
    MAX_RETRIES=30
    RETRY_COUNT=0
    
    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/health | grep -q "200"; then
            log_info "Backend is healthy"
            return 0
        fi
        
        RETRY_COUNT=$((RETRY_COUNT + 1))
        log_warn "Health check attempt $RETRY_COUNT/$MAX_RETRIES failed, retrying..."
        sleep 2
    done
    
    log_error "Health check failed after $MAX_RETRIES attempts"
    return 1
}

rollback() {
    log_info "Rolling back to previous deployment..."
    
    # Find the latest backup
    LATEST_BACKUP=$(ls -t "$BACKUP_DIR"/*.sql.gz 2>/dev/null | head -1)
    
    if [ -z "$LATEST_BACKUP" ]; then
        log_error "No backup found for rollback"
        exit 1
    fi
    
    log_warn "Rolling back database from: $LATEST_BACKUP"
    
    # Stop services
    docker-compose -f "$COMPOSE_FILE" down
    
    # Restore database
    gunzip < "$LATEST_BACKUP" | docker-compose -f "$COMPOSE_FILE" exec -T backend psql -U "$DB_USER" -h "$DB_HOST" "$DB_NAME"
    
    # Restart with previous images
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d
    
    log_info "Rollback completed"
}

show_status() {
    log_info "Service status:"
    docker-compose -f "$COMPOSE_FILE" ps
    
    log_info "Resource usage:"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"
}

cleanup() {
    log_info "Cleaning up old resources..."
    
    # Remove old images
    docker image prune -f
    
    # Remove old backups (keep last 7)
    ls -t "$BACKUP_DIR"/*.sql.gz 2>/dev/null | tail -n +8 | xargs -r rm
    
    log_info "Cleanup completed"
}

# =============================================================================
# Main
# =============================================================================

case "$1" in
    --build)
        check_prerequisites
        validate_env
        build_images
        deploy
        health_check
        ;;
    --migrate)
        check_prerequisites
        validate_env
        run_migrations
        ;;
    --rollback)
        check_prerequisites
        rollback
        ;;
    --status)
        show_status
        ;;
    --cleanup)
        cleanup
        ;;
    *)
        check_prerequisites
        validate_env
        backup_database
        deploy
        health_check
        cleanup
        log_info "Deployment successful!"
        ;;
esac
