#!/usr/bin/env bash
# backend.sh - Go backend process management functions
# Handles building, starting, stopping, and status checking for the Go server

# Note: _lib.sh is sourced by dev.sh before loading this file

# Backend configuration
APP_BIND_HOST="${APP_BIND_HOST:-127.0.0.1}"
APP_PUBLIC_HOST="${APP_PUBLIC_HOST:-$APP_BIND_HOST}"
BACKEND_HOST="${BACKEND_HOST:-$APP_BIND_HOST}"
BACKEND_PORT="${BACKEND_PORT:-37181}"
BACKEND_PUBLIC_HOST="${BACKEND_PUBLIC_HOST:-$APP_PUBLIC_HOST}"
BACKEND_PUBLIC_URL="${BACKEND_PUBLIC_URL:-http://$BACKEND_PUBLIC_HOST:$BACKEND_PORT}"
BACKEND_PID_FILE="$(get_pid_dir)/backend.pid"
BACKEND_LOG_FILE="$(get_log_dir)/backend.log"

#-------------------------------------------------------------------------------
# Backend process management
#-------------------------------------------------------------------------------

# backend_build - Build the Go backend server
backend_build() {
    local project_root
    project_root="$(get_project_root)"

    log_step "Building Go backend..."

    cd "$project_root/backend" || return 1

    # Build the server binary
    go build -o server ./cmd/server || {
        log_error "Failed to build backend"
        return 1
    }

    log_success "Backend built successfully"
    return 0
}

# backend_start - Start the Go backend server
# Builds first if needed, then starts the server and saves PID
backend_start() {
    local project_root
    project_root="$(get_project_root)"

    ensure_dev_dirs

    # Check if already running
    if pid_is_running "$BACKEND_PID_FILE"; then
        log_warn "Backend is already running (PID: $(cat "$BACKEND_PID_FILE"))"
        return 0
    fi

    log_step "Starting Go backend..."
    log_verbose "Working directory: $project_root/backend"

    cd "$project_root/backend" || return 1

    # Show environment being loaded
    show_env_vars "$project_root/.env"

    # Build first to ensure we have the binary
    if [ ! -f "server" ]; then
        log_verbose "Binary not found, building..."
        backend_build || return 1
    else
        log_verbose "Using existing binary"
    fi

    # Construct the start command for display
    # Source .env from project root (not backend directory)
    local frontend_url="${FRONTEND_PUBLIC_URL:-http://${APP_PUBLIC_HOST:-127.0.0.1}:${FRONTEND_PORT:-37180}}"
    local allowed_origins="${BACKEND_ALLOWED_ORIGINS:-$frontend_url}"
    local start_cmd="set -a && source \"$project_root/.env\" && set +a && HOST=\"$BACKEND_HOST\" PORT=\"$BACKEND_PORT\" FRONTEND_URL=\"$frontend_url\" OAUTH_REDIRECT_URL=\"$BACKEND_PUBLIC_URL\" ALLOWED_ORIGINS=\"$allowed_origins\" ./server"
    log_command "$start_cmd"

    # Start server in background with logging
    # Use env -S to load environment from .env file before starting server
    # This ensures the Go backend can access OAuth credentials via os.Getenv()
    log_verbose "Starting background process with nohup..."
    nohup env -i "$(which bash)" -c "$start_cmd" > "$BACKEND_LOG_FILE" 2>&1 &
    local backend_pid=$!

    log_verbose "Process started with PID: $backend_pid"

    # Save PID
    save_pid "$BACKEND_PID_FILE" "$backend_pid"

    # Wait a moment for the process to start
    log_verbose "Waiting for process to initialize..."
    sleep 2

    # Verify it's still running
    if ! pid_is_running "$BACKEND_PID_FILE"; then
        log_error "Backend failed to start. Check log: $BACKEND_LOG_FILE"
        log_verbose "Showing last 10 lines of log:"
        if [ -f "$BACKEND_LOG_FILE" ]; then
            tail -10 "$BACKEND_LOG_FILE" | sed 's/^/  /' >&2
        fi
        return 1
    fi

    log_verbose "Process is running, waiting for $BACKEND_HOST:$BACKEND_PORT to be ready..."

    # Wait for the port to be ready
    if ! wait_for_port "$BACKEND_PORT" 30 "$BACKEND_HOST"; then
        log_error "Backend did not start listening on $BACKEND_HOST:$BACKEND_PORT"
        log_error "Check log for errors: $BACKEND_LOG_FILE"
        kill_pid "$BACKEND_PID_FILE"
        return 1
    fi

    log_success "Backend started"
    echo -e "  ${C_DIM}PID:${C_RESET}       $backend_pid"
    echo -e "  ${C_DIM}Host:${C_RESET}      $BACKEND_HOST"
    echo -e "  ${C_DIM}Port:${C_RESET}      $BACKEND_PORT"
    echo -e "  ${C_DIM}Log:${C_RESET}       $BACKEND_LOG_FILE"
    echo -e "  ${C_DIM}Health:${C_RESET}    $BACKEND_PUBLIC_URL/health"

    return 0
}

# backend_stop - Gracefully stop the Go backend server
backend_stop() {
    log_step "Stopping Go backend..."

    if ! pid_is_running "$BACKEND_PID_FILE"; then
        log_warn "Backend is not running"
        # Clean up stale PID file
        rm -f "$BACKEND_PID_FILE" 2>/dev/null || true
        return 0
    fi

    local pid
    pid=$(cat "$BACKEND_PID_FILE")
    log_verbose "Found running process with PID: $pid"

    log_verbose "Sending SIGTERM (graceful shutdown)..."
    kill_pid "$BACKEND_PID_FILE" TERM

    # Give it a moment to shut down
    log_verbose "Waiting for graceful shutdown..."
    local count=0
    while [ $count -lt 5 ] && pid_is_running "$BACKEND_PID_FILE"; do
        sleep 1
        ((count++))
        echo -n "."
    done
    echo ""

    if pid_is_running "$BACKEND_PID_FILE"; then
        log_warn "Backend did not stop gracefully, forcing..."
        log_verbose "Sending SIGKILL (force quit)..."
        kill_pid "$BACKEND_PID_FILE" KILL
        sleep 1
    fi

    # Verify and cleanup
    if ! pid_is_running "$BACKEND_PID_FILE"; then
        log_success "Backend stopped"
        log_verbose "Cleaned up PID file"
    else
        log_error "Failed to stop backend (PID: $pid may be stuck)"
        return 1
    fi

    return 0
}

# backend_status - Check and display backend server status
backend_status() {
    echo ""
    echo "Backend Status:"
    echo "────────────────────────────────────────"

    if pid_is_running "$BACKEND_PID_FILE"; then
        local pid
        pid=$(cat "$BACKEND_PID_FILE")
        echo -e "${C_GREEN}●${C_RESET} Running (PID: $pid)"
        echo "  Host: $BACKEND_HOST"
        echo "  Port: $BACKEND_PORT"
        echo "  URL: $BACKEND_PUBLIC_URL"
        echo "  Log: $BACKEND_LOG_FILE"

        # Show memory usage if ps is available
        if command -v ps &>/dev/null; then
            local mem
            mem=$(ps -o rss= -p "$pid" 2>/dev/null | tr -d ' ')
            if [ -n "$mem" ]; then
                local mem_mb=$((mem / 1024))
                echo "  Memory: ${mem_mb}MB"
            fi
        fi
    else
        echo -e "${C_RED}○${C_RESET} Not running"
        # Clean up stale PID file
        rm -f "$BACKEND_PID_FILE" 2>/dev/null || true
    fi
    echo ""
}

# backend_restart - Restart the backend server
backend_restart() {
    log_step "Restarting backend..."
    backend_stop
    sleep 1
    backend_start
}

# backend_logs - Tail the backend log file
backend_logs() {
    if [ ! -f "$BACKEND_LOG_FILE" ]; then
        log_warn "No backend log file found"
        return 1
    fi

    log_info "Tailing backend logs (Ctrl+C to exit)..."
    tail -f "$BACKEND_LOG_FILE"
}

# backend_attach - Attach to the running backend process (for debugging)
# This shows the output as if the process was running in foreground
backend_attach() {
    if ! pid_is_running "$BACKEND_PID_FILE"; then
        log_error "Backend is not running"
        return 1
    fi

    log_info "Attaching to backend output (Ctrl+C to detach)..."
    tail -f "$BACKEND_LOG_FILE"
}
