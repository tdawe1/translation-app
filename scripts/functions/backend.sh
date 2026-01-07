#!/usr/bin/env bash
# backend.sh - Go backend process management functions
# Handles building, starting, stopping, and status checking for the Go server

# Note: _lib.sh is sourced by dev.sh before loading this file

# Backend configuration
BACKEND_PORT=8000
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

    cd "$project_root/backend" || return 1

    # Build first to ensure we have the binary
    if [ ! -f "server" ]; then
        backend_build || return 1
    fi

    # Start server in background with logging
    # Using nohup to ensure the process continues if shell exits
    nohup ./server > "$BACKEND_LOG_FILE" 2>&1 &
    local backend_pid=$!

    # Save PID
    save_pid "$BACKEND_PID_FILE" "$backend_pid"

    # Wait a moment for the process to start
    sleep 2

    # Verify it's still running
    if ! pid_is_running "$BACKEND_PID_FILE"; then
        log_error "Backend failed to start. Check log: $BACKEND_LOG_FILE"
        return 1
    fi

    # Wait for the port to be ready
    if ! wait_for_port "$BACKEND_PORT" 30; then
        log_error "Backend did not start listening on port $BACKEND_PORT"
        kill_pid "$BACKEND_PID_FILE"
        return 1
    fi

    log_success "Backend started (PID: $backend_pid, port: $BACKEND_PORT)"
    log_info "Log file: $BACKEND_LOG_FILE"
    return 0
}

# backend_stop - Gracefully stop the Go backend server
backend_stop() {
    log_step "Stopping Go backend..."

    if ! pid_is_running "$BACKEND_PID_FILE"; then
        log_warn "Backend is not running"
        return 0
    fi

    kill_pid "$BACKEND_PID_FILE" TERM

    # Give it a moment to shut down
    sleep 1

    if pid_is_running "$BACKEND_PID_FILE"; then
        log_warn "Backend did not stop gracefully, forcing..."
        kill_pid "$BACKEND_PID_FILE" KILL
    fi

    log_success "Backend stopped"
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
        echo "  Port: $BACKEND_PORT"
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
