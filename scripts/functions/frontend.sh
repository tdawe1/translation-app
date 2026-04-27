#!/usr/bin/env bash
# frontend.sh - Next.js frontend process management functions
# Handles starting, stopping, and status checking for the Next.js dev server

# Note: _lib.sh is sourced by dev.sh before loading this file

# Frontend configuration
APP_BIND_HOST="${APP_BIND_HOST:-127.0.0.1}"
APP_PUBLIC_HOST="${APP_PUBLIC_HOST:-$APP_BIND_HOST}"
FRONTEND_HOST="${FRONTEND_HOST:-$APP_BIND_HOST}"
FRONTEND_PORT="${FRONTEND_PORT:-37180}"
FRONTEND_PUBLIC_HOST="${FRONTEND_PUBLIC_HOST:-$APP_PUBLIC_HOST}"
FRONTEND_PUBLIC_URL="${FRONTEND_PUBLIC_URL:-http://$FRONTEND_PUBLIC_HOST:$FRONTEND_PORT}"
FRONTEND_API_URL="${FRONTEND_API_URL:-${NEXT_PUBLIC_API_URL:-${BACKEND_PUBLIC_URL:-http://${APP_PUBLIC_HOST:-127.0.0.1}:${BACKEND_PORT:-37181}}}}"
if [[ "$FRONTEND_API_URL" == https://* ]]; then
    DEFAULT_FRONTEND_WS_URL="wss://${FRONTEND_API_URL#https://}/ws"
else
    DEFAULT_FRONTEND_WS_URL="ws://${FRONTEND_API_URL#http://}/ws"
fi
FRONTEND_WS_URL="${FRONTEND_WS_URL:-${NEXT_PUBLIC_WS_URL:-$DEFAULT_FRONTEND_WS_URL}}"
FRONTEND_PID_FILE="$(get_pid_dir)/frontend.pid"
FRONTEND_LOG_FILE="$(get_log_dir)/frontend.log"

#-------------------------------------------------------------------------------
# Frontend process management
#-------------------------------------------------------------------------------

# frontend_install_deps - Install npm dependencies if needed
frontend_install_deps() {
    local project_root
    project_root="$(get_project_root)"

    cd "$project_root/frontend" || return 1

    if [ ! -d "node_modules" ]; then
        log_step "Installing frontend dependencies..."
        npm install || {
            log_error "Failed to install dependencies"
            return 1
        }
        log_success "Dependencies installed"
    fi
    return 0
}

# frontend_start - Start the Next.js frontend dev server
# Installs dependencies first if needed, then starts the server
frontend_start() {
    local project_root
    project_root="$(get_project_root)"

    ensure_dev_dirs

    # Check if already running
    if pid_is_running "$FRONTEND_PID_FILE"; then
        log_warn "Frontend is already running (PID: $(cat "$FRONTEND_PID_FILE"))"
        return 0
    fi

    log_step "Starting Next.js frontend..."
    log_verbose "Working directory: $project_root/frontend"
    log_verbose "Node version: $(node --version 2>/dev/null || echo 'not found')"
    log_verbose "npm version: $(npm --version 2>/dev/null || echo 'not found')"

    cd "$project_root/frontend" || return 1

    # Show frontend env being used
    show_env_vars ".env.local"

    # Ensure dependencies are installed
    if [ ! -d "node_modules" ]; then
        log_verbose "node_modules not found, installing dependencies..."
        frontend_install_deps || return 1
    else
        log_verbose "node_modules exists, skipping install"
    fi

    # Construct the start command for display
    local start_cmd="NEXT_PUBLIC_API_URL=\"$FRONTEND_API_URL\" NEXT_PUBLIC_WS_URL=\"$FRONTEND_WS_URL\" npm run dev -- --hostname \"$FRONTEND_HOST\" --port \"$FRONTEND_PORT\""
    log_command "$start_cmd"

    # Start dev server in background with logging
    log_verbose "Starting background process with nohup..."
    nohup env "PORT=$FRONTEND_PORT" "HOSTNAME=$FRONTEND_HOST" "NEXT_PUBLIC_API_URL=$FRONTEND_API_URL" "NEXT_PUBLIC_WS_URL=$FRONTEND_WS_URL" npm run dev -- --hostname "$FRONTEND_HOST" --port "$FRONTEND_PORT" > "$FRONTEND_LOG_FILE" 2>&1 &
    local frontend_pid=$!

    log_verbose "Process started with PID: $frontend_pid"

    # Save PID
    save_pid "$FRONTEND_PID_FILE" "$frontend_pid"

    # Wait for Next.js to start (can take a few seconds)
    log_verbose "Waiting for Next.js to initialize..."
    sleep 5

    # Verify it's still running
    if ! pid_is_running "$FRONTEND_PID_FILE"; then
        log_error "Frontend failed to start. Check log: $FRONTEND_LOG_FILE"
        log_verbose "Showing last 10 lines of log:"
        if [ -f "$FRONTEND_LOG_FILE" ]; then
            tail -10 "$FRONTEND_LOG_FILE" | sed 's/^/  /' >&2
        fi
        return 1
    fi

    log_verbose "Process is running, waiting for $FRONTEND_HOST:$FRONTEND_PORT to be ready..."

    # Wait for the port to be ready (Next.js can take 10-15s on first start)
    if ! wait_for_port "$FRONTEND_PORT" 30 "$FRONTEND_HOST"; then
        log_error "Frontend did not start listening on $FRONTEND_HOST:$FRONTEND_PORT"
        log_error "Check log for errors: $FRONTEND_LOG_FILE"
        kill_pid "$FRONTEND_PID_FILE"
        return 1
    fi

    log_success "Frontend started"
    echo -e "  ${C_DIM}PID:${C_RESET}       $frontend_pid"
    echo -e "  ${C_DIM}Host:${C_RESET}      $FRONTEND_HOST"
    echo -e "  ${C_DIM}Port:${C_RESET}      $FRONTEND_PORT"
    echo -e "  ${C_DIM}Log:${C_RESET}       $FRONTEND_LOG_FILE"
    echo -e "  ${C_DIM}URL:${C_RESET}        $FRONTEND_PUBLIC_URL"
    echo -e "  ${C_DIM}API:${C_RESET}        $FRONTEND_API_URL"

    return 0
}

# frontend_stop - Stop the Next.js frontend server
frontend_stop() {
    log_step "Stopping Next.js frontend..."

    if ! pid_is_running "$FRONTEND_PID_FILE"; then
        log_warn "Frontend is not running"
        # Clean up stale PID file
        rm -f "$FRONTEND_PID_FILE" 2>/dev/null || true
        return 0
    fi

    local pid
    pid=$(cat "$FRONTEND_PID_FILE")
    log_verbose "Found running process with PID: $pid"

    log_verbose "Sending SIGTERM (graceful shutdown)..."
    kill_pid "$FRONTEND_PID_FILE" TERM

    # Give it a moment to shut down
    log_verbose "Waiting for graceful shutdown..."
    local count=0
    while [ $count -lt 5 ] && pid_is_running "$FRONTEND_PID_FILE"; do
        sleep 1
        ((count++))
        echo -n "."
    done
    echo ""

    if pid_is_running "$FRONTEND_PID_FILE"; then
        log_warn "Frontend did not stop gracefully, forcing..."
        log_verbose "Sending SIGKILL (force quit)..."
        kill_pid "$FRONTEND_PID_FILE" KILL
        sleep 1
    fi

    # Verify and cleanup
    if ! pid_is_running "$FRONTEND_PID_FILE"; then
        log_success "Frontend stopped"
        log_verbose "Cleaned up PID file"
    else
        log_error "Failed to stop frontend (PID: $pid may be stuck)"
        return 1
    fi

    return 0
}

# frontend_status - Check and display frontend server status
frontend_status() {
    echo ""
    echo "Frontend Status:"
    echo "────────────────────────────────────────"

    if pid_is_running "$FRONTEND_PID_FILE"; then
        local pid
        pid=$(cat "$FRONTEND_PID_FILE")
        echo -e "${C_GREEN}●${C_RESET} Running (PID: $pid)"
        echo "  Host: $FRONTEND_HOST"
        echo "  Port: $FRONTEND_PORT"
        echo "  URL: $FRONTEND_PUBLIC_URL"
        echo "  API: $FRONTEND_API_URL"
        echo "  Log: $FRONTEND_LOG_FILE"

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
        rm -f "$FRONTEND_PID_FILE" 2>/dev/null || true
    fi
    echo ""
}

# frontend_restart - Restart the frontend server
frontend_restart() {
    log_step "Restarting frontend..."
    frontend_stop
    sleep 2
    frontend_start
}

# frontend_logs - Tail the frontend log file
frontend_logs() {
    if [ ! -f "$FRONTEND_LOG_FILE" ]; then
        log_warn "No frontend log file found"
        return 1
    fi

    log_info "Tailing frontend logs (Ctrl+C to exit)..."
    tail -f "$FRONTEND_LOG_FILE"
}

# frontend_build - Create a production build of the frontend
frontend_build() {
    local project_root
    project_root="$(get_project_root)"

    log_step "Building frontend for production..."

    cd "$project_root/frontend" || return 1

    # Ensure dependencies are installed
    frontend_install_deps || return 1

    # Run production build
    npm run build || {
        log_error "Frontend build failed"
        return 1
    }

    log_success "Frontend build complete"
    return 0
}
