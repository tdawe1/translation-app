#!/usr/bin/env bash
# dev.sh - Development environment controller
# Main entry point for starting/stopping the full development stack

set -e

# Source shared library and all function files
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_lib.sh"
source "$SCRIPT_DIR/functions/docker.sh"
source "$SCRIPT_DIR/functions/backend.sh"
source "$SCRIPT_DIR/functions/frontend.sh"
source "$SCRIPT_DIR/functions/logs.sh"

#-------------------------------------------------------------------------------
# Usage information
#-------------------------------------------------------------------------------

show_usage() {
    cat <<'EOF'
Usage: ./dev.sh <command> [options]

Development environment controller for GengoWatcher SaaS.

Commands:
  up                    Start all services (docker → backend → frontend)
  down                  Stop all services (frontend → backend → docker)
  restart               Restart all services
  status                Show status of all services
  logs [service]        Tail logs from all or specific services

  backend <cmd>         Manage backend: start|stop|restart|status|logs
  frontend <cmd>        Manage frontend: start|stop|restart|status|logs
  docker <cmd>          Manage docker: start|stop|restart|status|logs

  logs list             List all available log files
  logs clear            Clear all log files

  check                 Validate development environment

Examples:
  ./dev.sh up                    # Start everything
  ./dev.sh down                  # Stop everything
  ./dev.sh logs backend          # Watch backend logs
  ./dev.sh backend restart       # Restart only the backend
  ./dev.sh status                # Check all service statuses

EOF
}

#-------------------------------------------------------------------------------
# Main commands
#-------------------------------------------------------------------------------

# cmd_up - Start all services in dependency order
cmd_up() {
    log_step "Starting development environment..."
    echo ""

    # Validate environment first
    if ! check_environment; then
        log_error "Environment validation failed. Aborting."
        return 1
    fi

    # Start Docker services
    if ! docker_up; then
        log_error "Failed to start Docker services"
        return 1
    fi
    echo ""

    # Start backend
    if ! backend_start; then
        log_error "Failed to start backend"
        return 1
    fi
    echo ""

    # Start frontend
    if ! frontend_start; then
        log_error "Failed to start frontend"
        return 1
    fi
    echo ""

    # Show summary
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_success "Development environment is ready!"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "  Services:"
    echo "    • PostgreSQL:  localhost:5433"
    echo "    • Redis:      localhost:6380"
    echo "    • MailHog:    http://localhost:8025"
    echo "    • Backend:    http://localhost:8000"
    echo "    • Frontend:   http://localhost:3001"
    echo ""
    echo "  Commands:"
    echo "    ./dev.sh logs              # View all logs"
    echo "    ./dev.sh status            # Check service status"
    echo "    ./dev.sh down              # Stop everything"
    echo ""
}

# cmd_down - Stop all services in reverse dependency order
cmd_down() {
    log_step "Stopping development environment..."
    echo ""

    # Stop frontend first
    frontend_stop
    echo ""

    # Stop backend
    backend_stop
    echo ""

    # Stop Docker services
    docker_down
    echo ""

    # Fallback: Kill any orphaned processes on managed ports
    cleanup_ports
    echo ""

    log_success "Development environment stopped"
    echo ""
}

# cmd_restart - Restart all services
cmd_restart() {
    log_step "Restarting development environment..."
    echo ""

    cmd_down
    sleep 2
    echo ""

    cmd_up
}

# cmd_status - Show status of all services
cmd_status() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Development Environment Status"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    # Docker status
    docker_status

    # Backend status
    backend_status

    # Frontend status
    frontend_status

    # Overall summary
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    local all_running=true

    if ! docker_status | grep -q "healthy\|running\|Up"; then
        all_running=false
    fi

    if ! pid_is_running "$(get_pid_dir)/backend.pid"; then
        all_running=false
    fi

    if ! pid_is_running "$(get_pid_dir)/frontend.pid"; then
        all_running=false
    fi

    if [ "$all_running" = true ]; then
        echo -e "${C_GREEN}● All services running${C_RESET}"
    else
        echo -e "${C_YELLOW}○ Some services not running${C_RESET}"
    fi
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
}

# cmd_logs - Show logs from all or specific services
cmd_logs() {
    local service="$1"

    if [ -z "$service" ]; then
        logs_all
    else
        logs_service "$service"
    fi
}

# cmd_backend - Handle backend-specific commands
cmd_backend() {
    local subcmd="$1"

    case "$subcmd" in
        start)
            backend_start
            ;;
        stop)
            backend_stop
            ;;
        restart)
            backend_restart
            ;;
        status)
            backend_status
            ;;
        logs)
            backend_logs
            ;;
        *)
            log_error "Unknown backend command: $subcmd"
            echo ""
            echo "Available backend commands:"
            echo "  start    - Start the backend server"
            echo "  stop     - Stop the backend server"
            echo "  restart  - Restart the backend server"
            echo "  status   - Show backend status"
            echo "  logs     - Tail backend logs"
            echo ""
            return 1
            ;;
    esac
}

# cmd_frontend - Handle frontend-specific commands
cmd_frontend() {
    local subcmd="$1"

    case "$subcmd" in
        start)
            frontend_start
            ;;
        stop)
            frontend_stop
            ;;
        restart)
            frontend_restart
            ;;
        status)
            frontend_status
            ;;
        logs)
            frontend_logs
            ;;
        build)
            frontend_build
            ;;
        *)
            log_error "Unknown frontend command: $subcmd"
            echo ""
            echo "Available frontend commands:"
            echo "  start    - Start the frontend server"
            echo "  stop     - Stop the frontend server"
            echo "  restart  - Restart the frontend server"
            echo "  status   - Show frontend status"
            echo "  logs     - Tail frontend logs"
            echo "  build    - Create production build"
            echo ""
            return 1
            ;;
    esac
}

# cmd_docker - Handle Docker-specific commands
cmd_docker() {
    local subcmd="$1"

    case "$subcmd" in
        start)
            docker_up
            ;;
        stop)
            docker_down
            ;;
        restart)
            docker_restart
            ;;
        status)
            docker_status
            ;;
        logs)
            docker_logs
            ;;
        *)
            log_error "Unknown docker command: $subcmd"
            echo ""
            echo "Available docker commands:"
            echo "  start    - Start Docker services"
            echo "  stop     - Stop Docker services"
            echo "  restart  - Restart Docker services"
            echo "  status   - Show Docker service status"
            echo "  logs     - Tail Docker logs"
            echo ""
            return 1
            ;;
    esac
}

# cmd_logs_list - List all log files
cmd_logs_list() {
    logs_list
}

# cmd_logs_clear - Clear all log files
cmd_logs_clear() {
    logs_clear
}

# cmd_check - Validate development environment
cmd_check() {
    if check_environment; then
        log_success "Environment check passed"
        return 0
    else
        log_error "Environment check failed"
        return 1
    fi
}

#-------------------------------------------------------------------------------
# Main entry point
#-------------------------------------------------------------------------------

main() {
    # Ensure .dev directory structure exists
    ensure_dev_dirs

    # Parse command
    local cmd="$1"
    shift || true

    case "$cmd" in
        up)
            cmd_up
            ;;
        down)
            cmd_down
            ;;
        restart)
            cmd_restart
            ;;
        status)
            cmd_status
            ;;
        logs)
            cmd_logs "$1"
            ;;
        backend)
            cmd_backend "$1"
            ;;
        frontend)
            cmd_frontend "$1"
            ;;
        docker)
            cmd_docker "$1"
            ;;
        check)
            cmd_check
            ;;
        help|--help|-h)
            show_usage
            ;;
        "")
            show_usage
            ;;
        *)
            log_error "Unknown command: $cmd"
            echo ""
            show_usage
            exit 1
            ;;
    esac
}

# Run main if this script is executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
