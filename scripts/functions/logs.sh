#!/usr/bin/env bash
# logs.sh - Log aggregation and viewing functions
# Provides unified log viewing across all services

# Note: _lib.sh is sourced by dev.sh before loading this file

#-------------------------------------------------------------------------------
# Log file paths
#-------------------------------------------------------------------------------

BACKEND_LOG="$(get_log_dir)/backend.log"
FRONTEND_LOG="$(get_log_dir)/frontend.log"

#-------------------------------------------------------------------------------
# Log aggregation functions
#-------------------------------------------------------------------------------

# logs_all - Show combined logs from all services with service labels
logs_all() {
    local project_root
    local compose_cmd
    project_root="$(get_project_root)"
    compose_cmd="$(get_docker_compose_cmd)"

    log_info "Tailing all service logs (Ctrl+C to exit)..."
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Watching logs from all services"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    # Use tail -F to follow all log files, including ones that might be created later
    local tail_files=()

    # Add backend log if it exists or create placeholder
    if [ -f "$BACKEND_LOG" ]; then
        tail_files+=("$BACKEND_LOG")
    fi

    # Add frontend log if it exists or create placeholder
    if [ -f "$FRONTEND_LOG" ]; then
        tail_files+=("$FRONTEND_LOG")
    fi

    # Start tailing files we have, plus docker logs
    if [ ${#tail_files[@]} -gt 0 ]; then
        # Run in background so we can also capture docker logs
        (
            tail -F "${tail_files[@]}" 2>/dev/null | sed 's/^/[APP] /' &
            tail_pid=$!
            trap "kill $tail_pid 2>/dev/null || true" EXIT
        ) &
    fi

    # Also tail docker logs
    cd "$project_root" || return 1
    $compose_cmd logs -f 2>&1 | sed 's/^/[DOCKER] /'
}

# logs_backend - Tail only the backend log file with syntax highlighting
logs_backend() {
    if [ ! -f "$BACKEND_LOG" ]; then
        log_warn "Backend log file not found: $BACKEND_LOG"
        log_info "Backend may not be running yet. Start it with: ./dev.sh backend start"
        return 1
    fi

    log_info "Tailing backend logs (Ctrl+C to exit)..."
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Backend Log: $BACKEND_LOG"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    tail -F "$BACKEND_LOG"
}

# logs_frontend - Tail only the frontend log file with syntax highlighting
logs_frontend() {
    if [ ! -f "$FRONTEND_LOG" ]; then
        log_warn "Frontend log file not found: $FRONTEND_LOG"
        log_info "Frontend may not be running yet. Start it with: ./dev.sh frontend start"
        return 1
    fi

    log_info "Tailing frontend logs (Ctrl+C to exit)..."
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Frontend Log: $FRONTEND_LOG"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    tail -F "$FRONTEND_LOG"
}

# logs_docker - Tail only Docker Compose logs
logs_docker() {
    local project_root
    local compose_cmd
    local service="$1"
    project_root="$(get_project_root)"
    compose_cmd="$(get_docker_compose_cmd)"

    cd "$project_root" || return 1

    if [ -n "$service" ]; then
        log_info "Tailing Docker logs for '$service' (Ctrl+C to exit)..."
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "  Docker Service: $service"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        $compose_cmd logs -f "$service"
    else
        log_info "Tailing Docker logs for all services (Ctrl+C to exit)..."
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "  Docker Compose Services"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        $compose_cmd logs -f
    fi
}

# logs_service - Generic function to tail logs for a specific service
# Usage: logs_service <backend|frontend|docker|postgres|redis|mailhog>
logs_service() {
    local service="$1"

    case "$service" in
        backend)
            logs_backend
            ;;
        frontend)
            logs_frontend
            ;;
        docker)
            logs_docker
            ;;
        postgres)
            logs_docker postgres
            ;;
        redis)
            logs_docker redis
            ;;
        mailhog)
            logs_docker mailhog
            ;;
        *)
            log_error "Unknown service: $service"
            echo ""
            echo "Available services:"
            echo "  - backend    : Go backend server logs"
            echo "  - frontend   : Next.js frontend logs"
            echo "  - docker     : All Docker Compose logs"
            echo "  - postgres   : PostgreSQL container logs"
            echo "  - redis      : Redis container logs"
            echo "  - mailhog    : MailHog container logs"
            echo ""
            return 1
            ;;
    esac
}

# logs_list - List all available log files
logs_list() {
    local log_dir
    log_dir="$(get_log_dir)"

    echo ""
    echo "Log Files:"
    echo "────────────────────────────────────────"

    if [ -d "$log_dir" ]; then
        for log_file in "$log_dir"/*; do
            if [ -f "$log_file" ]; then
                local filename
                local size
                local mtime
                filename=$(basename "$log_file")
                size=$(du -h "$log_file" | cut -f1)
                mtime=$(stat -c "%y" "$log_file" 2>/dev/null | cut -d'.' -f1 | cut -d' ' -f1,2)
                printf "  %-20s %8s  %s\n" "$filename" "$size" "$mtime"
            fi
        done
    else
        echo "  (no log files found)"
    fi
    echo ""
}

# logs_clear - Remove all log files
logs_clear() {
    local log_dir
    log_dir="$(get_log_dir)"

    if [ ! -d "$log_dir" ]; then
        log_info "No log directory found"
        return 0
    fi

    log_warn "This will delete all log files in $log_dir"
    read -p "Continue? (y/N) " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -f "$log_dir"/*
        log_success "Log files cleared"
    else
        log_info "Cancelled"
    fi
}
