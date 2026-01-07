#!/usr/bin/env bash
# _lib.sh - Shared functions for all dev scripts
# Provides logging, helpers, and common utilities

#-------------------------------------------------------------------------------
# Colors and terminal output
#-------------------------------------------------------------------------------

# ANSI color codes (use \033 for portability)
C_RESET='\033[0m'
C_BOLD='\033[1m'
C_DIM='\033[2m'
C_RED='\033[31m'
C_GREEN='\033[32m'
C_YELLOW='\033[33m'
C_BLUE='\033[34m'
C_MAGENTA='\033[35m'
C_CYAN='\033[36m'
C_WHITE='\033[37m'

# log_info - Print info message in blue
log_info() {
    echo -e "${C_BLUE}[INFO]${C_RESET} $*"
}

# log_success - Print success message in green
log_success() {
    echo -e "${C_GREEN}[OK]${C_RESET} $*"
}

# log_error - Print error message in red
log_error() {
    echo -e "${C_RED}[ERROR]${C_RESET} $*" >&2
}

# log_warn - Print warning message in yellow
log_warn() {
    echo -e "${C_YELLOW}[WARN]${C_RESET} $*"
}

# log_step - Print step header in bold cyan
log_step() {
    echo -e "${C_BOLD}${C_CYAN}==>${C_RESET} $*"
}

#-------------------------------------------------------------------------------
# Project paths
#-------------------------------------------------------------------------------

# get_project_root - Return absolute path to project root
# Uses the location of this script file to determine project root
get_project_root() {
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    printf '%s' "$(cd "$script_dir/.." && pwd)"
}

# get_pid_dir - Return path to PID storage directory
get_pid_dir() {
    printf '%s/pids' "$(get_project_root)/.dev"
}

# get_log_dir - Return path to log storage directory
get_log_dir() {
    printf '%s/logs' "$(get_project_root)/.dev"
}

# ensure_dev_dirs - Create .dev directory structure if missing
ensure_dev_dirs() {
    mkdir -p "$(get_pid_dir)"
    mkdir -p "$(get_log_dir)"
}

#-------------------------------------------------------------------------------
# Command validation
#-------------------------------------------------------------------------------

# require_cmd - Exit if command is not available
# Usage: require_cmd docker
require_cmd() {
    local cmd="$1"
    if ! command -v "$cmd" &>/dev/null; then
        log_error "Required command '$cmd' not found. Please install it first."
        exit 1
    fi
}

# require_cmds - Check multiple commands at once
# Usage: require_cmds docker go node
require_cmds() {
    local missing=()
    for cmd in "$@"; do
        if ! command -v "$cmd" &>/dev/null; then
            missing+=("$cmd")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Missing required commands: ${missing[*]}"
        exit 1
    fi
}

#-------------------------------------------------------------------------------
# Service health checks
#-------------------------------------------------------------------------------

# wait_for_service - Poll until service responds or timeout
# Usage: wait_for_service <url> <timeout_seconds>
# Returns: 0 if service is up, 1 if timeout
wait_for_service() {
    local url="$1"
    local timeout="${2:-30}"
    local start_time
    start_time=$(date +%s)

    log_info "Waiting for service at $url (timeout: ${timeout}s)..."

    while true; do
        local current_time
        current_time=$(date +%s)
        local elapsed=$((current_time - start_time))

        if [ $elapsed -ge $timeout ]; then
            log_error "Timeout waiting for service at $url"
            return 1
        fi

        # Try to connect using curl
        if command -v curl &>/dev/null; then
            if curl -sSf "$url" &>/dev/null; then
                return 0
            fi
        # Fallback to wget if curl not available
        elif command -v wget &>/dev/null; then
            if wget -q -O /dev/null "$url" 2>/dev/null; then
                return 0
            fi
        # Fallback to netcat for TCP port checks
        elif command -v nc &>/dev/null; then
            # Extract host:port from URL
            local host_port
            host_port="${url#*://}"
            host_port="${host_port%%/*}"

            if nc -z "${host_port%:*}" "${host_port#*:}" 2>/dev/null; then
                return 0
            fi
        fi

        sleep 1
    done
}

# wait_for_port - Wait until a port is accepting connections
# Usage: wait_for_port <port> <timeout_seconds>
wait_for_port() {
    local port="$1"
    local timeout="${2:-30}"
    local start_time
    start_time=$(date +%s)

    log_info "Waiting for port $port (timeout: ${timeout}s)..."

    while true; do
        local current_time
        current_time=$(date +%s)
        local elapsed=$((current_time - start_time))

        if [ $elapsed -ge $timeout ]; then
            log_error "Timeout waiting for port $port"
            return 1
        fi

        if command -v nc &>/dev/null; then
            if nc -z 127.0.0.1 "$port" 2>/dev/null; then
                return 0
            fi
        elif command -v bash &>/dev/null; then
            # Bash built-in TCP check
            if timeout 1 bash -c "cat < /dev/null > /dev/tcp/127.0.0.1/$port" 2>/dev/null; then
                return 0
            fi
        fi

        sleep 1
    done
}

#-------------------------------------------------------------------------------
# Process management helpers
#-------------------------------------------------------------------------------

# pid_is_running - Check if a PID is currently running
# Usage: pid_is_running <pid_file>
# Returns: 0 if running, 1 if not
pid_is_running() {
    local pid_file="$1"
    local pid

    if [ ! -f "$pid_file" ]; then
        return 1
    fi

    pid=$(cat "$pid_file" 2>/dev/null)

    # Check if PID is a number
    if ! [[ "$pid" =~ ^[0-9]+$ ]]; then
        return 1
    fi

    # Check if process is running
    if kill -0 "$pid" 2>/dev/null; then
        return 0
    fi

    # PID file exists but process is not running - clean it up
    rm -f "$pid_file"
    return 1
}

# save_pid - Save process ID to file
# Usage: save_pid <pid_file> <pid>
save_pid() {
    local pid_file="$1"
    local pid="$2"
    printf '%s' "$pid" > "$pid_file"
}

# kill_pid - Gracefully kill process and remove PID file
# Usage: kill_pid <pid_file> [signal]
kill_pid() {
    local pid_file="$1"
    local signal="${2:-TERM}"
    local pid

    if [ ! -f "$pid_file" ]; then
        return 0
    fi

    pid=$(cat "$pid_file" 2>/dev/null)

    if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
        kill -"$signal" "$pid" 2>/dev/null || true

        # Wait for graceful shutdown, then force kill if needed
        local count=0
        while kill -0 "$pid" 2>/dev/null && [ $count -lt 5 ]; do
            sleep 1
            ((count++))
        done

        if kill -0 "$pid" 2>/dev/null; then
            kill -KILL "$pid" 2>/dev/null || true
        fi
    fi

    rm -f "$pid_file"
}

#-------------------------------------------------------------------------------
# Port-based cleanup (fallback for orphaned processes)
#-------------------------------------------------------------------------------

# cleanup_ports - Kill any process listening on managed ports
# Usage: cleanup_ports
# This is a fallback cleanup that doesn't rely on PID files
# Returns: 0 always (errors are logged but don't fail)
cleanup_ports() {
    local ports=(8000 3001)  # Managed by our processes (Docker handles its own)
    local killed_any=0

    log_info "Checking for orphaned processes on managed ports..."

    for port in "${ports[@]}"; do
        local pids=()

        # Find PIDs listening on this port (works with lsof, ss, netstat)
        if command -v lsof &>/dev/null; then
            # lsof -t outputs only PIDs
            mapfile -t pids < <(lsof -ti ":$port" 2>/dev/null)
        elif command -v ss &>/dev/null; then
            # ss format: extract PID from ss output
            pids=($(ss -tlnp 2>/dev/null | grep ":$port " | grep -oP 'pid=\K\d+'))
        fi

        if [ ${#pids[@]} -eq 0 ]; then
            continue  # No process on this port
        fi

        for pid in "${pids[@]}"; do
            if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
                local cmdline
                cmdline=$(ps -p "$pid" -o args= 2>/dev/null || echo "unknown")

                log_warn "Killing orphaned process on port $port (PID: $pid, $cmdline)"

                kill -TERM "$pid" 2>/dev/null || true
                killed_any=1

                # Wait for graceful shutdown (max 3 seconds)
                local count=0
                while kill -0 "$pid" 2>/dev/null && [ $count -lt 3 ]; do
                    sleep 1
                    ((count++))
                done

                # Force kill if still running
                if kill -0 "$pid" 2>/dev/null; then
                    kill -KILL "$pid" 2>/dev/null || true
                    log_warn "Force killed PID $pid on port $port"
                fi
            fi
        done
    done

    if [ $killed_any -eq 1 ]; then
        # Extra wait to ensure ports are fully released
        sleep 1
    fi

    return 0
}

#-------------------------------------------------------------------------------
# Port availability checks
#-------------------------------------------------------------------------------

# is_port_in_use - Check if a port is already in use
# Usage: is_port_in_use <port>
# Returns: 0 if in use, 1 if available
is_port_in_use() {
    local port="$1"

    if command -v lsof &>/dev/null; then
        lsof -i ":$port" &>/dev/null
        return $?
    elif command -v netstat &>/dev/null; then
        netstat -an 2>/dev/null | grep -q "[.:]$port .*LISTEN"
        return $?
    elif command -v ss &>/dev/null; then
        ss -ltn 2>/dev/null | grep -q ":$port "
        return $?
    fi

    # Fallback: try to bind to the port
    if command -v python3 &>/dev/null; then
        python3 -c "import socket; s=socket.socket(); s.bind(('127.0.0.1',$port)); s.close()" 2>/dev/null
        if [ $? -eq 0 ]; then
            return 1  # Port is available
        fi
        return 0  # Port is in use
    fi

    return 1  # Assume in use if we can't check
}

# check_required_ports - Verify all required ports are available
# Usage: check_required_ports
check_required_ports() {
    local ports=(5433 6380 8000 3001)
    local conflicts=()

    for port in "${ports[@]}"; do
        if is_port_in_use "$port"; then
            conflicts+=("$port")
        fi
    done

    if [ ${#conflicts[@]} -gt 0 ]; then
        log_error "Required ports already in use: ${conflicts[*]}"
        log_error "Please free these ports before starting dev environment."
        return 1
    fi

    return 0
}

#-------------------------------------------------------------------------------
# Environment validation
#-------------------------------------------------------------------------------

# check_environment - Validate development environment
# Returns: 0 if all checks pass, 1 otherwise
check_environment() {
    local project_root
    project_root="$(get_project_root)"
    local has_errors=0

    log_step "Validating development environment..."

    # Check Docker
    if ! command -v docker &>/dev/null; then
        log_error "Docker not found. Please install Docker."
        ((has_errors++))
    elif ! docker info &>/dev/null; then
        log_error "Docker is not running. Please start Docker."
        ((has_errors++))
    fi

    # Check docker-compose
    if ! command -v docker-compose &>/dev/null && ! docker compose version &>/dev/null; then
        log_error "docker-compose not found. Please install Docker Compose."
        ((has_errors++))
    fi

    # Check Go
    if ! command -v go &>/dev/null; then
        log_error "Go not found. Please install Go 1.23+."
        ((has_errors++))
    fi

    # Check Node.js
    if ! command -v node &>/dev/null; then
        log_error "Node.js not found. Please install Node.js 18+."
        ((has_errors++))
    fi

    # Check .env file
    if [ ! -f "$project_root/.env" ]; then
        if [ -f "$project_root/.env.example" ]; then
            log_warn ".env not found. Creating from .env.example..."
            cp "$project_root/.env.example" "$project_root/.env"
            log_warn "Please edit .env with your configuration."
        else
            log_error ".env not found and no .env.example available."
            ((has_errors++))
        fi
    fi

    # Check port availability
    if ! check_required_ports; then
        ((has_errors++))
    fi

    if [ $has_errors -gt 0 ]; then
        log_error "Environment validation failed. Please fix the errors above."
        return 1
    fi

    log_success "Environment validation passed."
    return 0
}

#-------------------------------------------------------------------------------
# Version detection helpers
#-------------------------------------------------------------------------------

# get_docker_compose_cmd - Return the correct docker-compose command
# Handles both 'docker-compose' (v1) and 'docker compose' (v2)
get_docker_compose_cmd() {
    if docker compose version &>/dev/null; then
        printf 'docker compose'
    elif command -v docker-compose &>/dev/null; then
        printf 'docker-compose'
    else
        log_error "docker-compose not found"
        return 1
    fi
}
