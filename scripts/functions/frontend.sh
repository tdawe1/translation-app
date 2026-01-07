#!/usr/bin/env bash
# frontend.sh - Next.js frontend process management functions
# Handles starting, stopping, and status checking for the Next.js dev server

# Note: _lib.sh is sourced by dev.sh before loading this file

# Frontend configuration
FRONTEND_PORT=3001
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

    cd "$project_root/frontend" || return 1

    # Ensure dependencies are installed
    frontend_install_deps || return 1

    # Start dev server in background with logging
    # Using nohup to ensure the process continues if shell exits
    # PORT env var tells Next.js which port to listen on
    nohup env "PORT=$FRONTEND_PORT" npm run dev > "$FRONTEND_LOG_FILE" 2>&1 &
    local frontend_pid=$!

    # Save PID
    save_pid "$FRONTEND_PID_FILE" "$frontend_pid"

    # Wait for Next.js to start (can take a few seconds)
    log_info "Waiting for Next.js to start..."
    sleep 5

    # Verify it's still running
    if ! pid_is_running "$FRONTEND_PID_FILE"; then
        log_error "Frontend failed to start. Check log: $FRONTEND_LOG_FILE"
        return 1
    fi

    # Wait for the port to be ready (Next.js can take 10-15s on first start)
    if ! wait_for_port "$FRONTEND_PORT" 30; then
        log_error "Frontend did not start listening on port $FRONTEND_PORT"
        log_error "Check log for errors: $FRONTEND_LOG_FILE"
        kill_pid "$FRONTEND_PID_FILE"
        return 1
    fi

    log_success "Frontend started (PID: $frontend_pid, port: $FRONTEND_PORT)"
    log_info "Log file: $FRONTEND_LOG_FILE"
    log_info "Visit: http://localhost:$FRONTEND_PORT"
    return 0
}

# frontend_stop - Stop the Next.js frontend server
frontend_stop() {
    log_step "Stopping Next.js frontend..."

    if ! pid_is_running "$FRONTEND_PID_FILE"; then
        log_warn "Frontend is not running"
        return 0
    fi

    kill_pid "$FRONTEND_PID_FILE" TERM

    # Give it a moment to shut down
    sleep 2

    if pid_is_running "$FRONTEND_PID_FILE"; then
        log_warn "Frontend did not stop gracefully, forcing..."
        kill_pid "$FRONTEND_PID_FILE" KILL
    fi

    log_success "Frontend stopped"
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
        echo "  Port: $FRONTEND_PORT"
        echo "  URL: http://localhost:$FRONTEND_PORT"
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
