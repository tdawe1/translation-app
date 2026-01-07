#!/usr/bin/env bash
# docker.sh - Docker Compose service management functions
# Handles PostgreSQL, Redis, and MailHog containers

# Note: _lib.sh is sourced by dev.sh before loading this file

#-------------------------------------------------------------------------------
# Docker service management
#-------------------------------------------------------------------------------

# docker_up - Start Docker Compose services in detached mode
# Waits for health checks to pass before returning
docker_up() {
    local project_root
    local compose_cmd
    project_root="$(get_project_root)"
    compose_cmd="$(get_docker_compose_cmd)"

    log_step "Starting Docker services..."

    cd "$project_root" || return 1

    # Start services
    $compose_cmd up -d || {
        log_error "Failed to start Docker services"
        return 1
    }

    log_info "Waiting for services to become healthy..."

    # Wait for PostgreSQL (max 30 seconds)
    log_info "Waiting for PostgreSQL..."
    local count=0
    while [ $count -lt 30 ]; do
        if $compose_cmd ps postgres | grep -q "healthy"; then
            log_success "PostgreSQL is healthy"
            break
        fi
        sleep 1
        ((count++))
    done

    if [ $count -ge 30 ]; then
        log_warn "PostgreSQL health check timeout, but continuing..."
    fi

    # Wait for Redis (max 15 seconds)
    log_info "Waiting for Redis..."
    count=0
    while [ $count -lt 15 ]; do
        if $compose_cmd ps redis | grep -q "healthy\|running"; then
            log_success "Redis is running"
            break
        fi
        sleep 1
        ((count++))
    done

    if [ $count -ge 15 ]; then
        log_warn "Redis health check timeout, but continuing..."
    fi

    log_success "Docker services started"
    return 0
}

# docker_down - Stop and remove Docker Compose services
docker_down() {
    local project_root
    local compose_cmd
    project_root="$(get_project_root)"
    compose_cmd="$(get_docker_compose_cmd)"

    log_step "Stopping Docker services..."

    cd "$project_root" || return 1

    $compose_cmd down || {
        log_error "Failed to stop Docker services"
        return 1
    }

    log_success "Docker services stopped"
    return 0
}

# docker_status - Show status of all Docker services
docker_status() {
    local project_root
    local compose_cmd
    project_root="$(get_project_root)"
    compose_cmd="$(get_docker_compose_cmd)"

    cd "$project_root" || return 1

    echo ""
    echo "Docker Services Status:"
    echo "────────────────────────────────────────"
    $compose_cmd ps
    echo ""
}

# docker_logs - Tail logs from Docker services
# Usage: docker_logs [service_name]
docker_logs() {
    local project_root
    local compose_cmd
    local service="$1"
    project_root="$(get_project_root)"
    compose_cmd="$(get_docker_compose_cmd)"

    cd "$project_root" || return 1

    if [ -n "$service" ]; then
        log_info "Tailing logs for $service (Ctrl+C to exit)..."
        $compose_cmd logs -f "$service"
    else
        log_info "Tailing logs for all services (Ctrl+C to exit)..."
        $compose_cmd logs -f
    fi
}

# docker_restart - Restart Docker services
docker_restart() {
    log_step "Restarting Docker services..."
    docker_down
    sleep 2
    docker_up
}
