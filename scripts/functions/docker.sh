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
	log_verbose "Working directory: $project_root"
	log_verbose "Docker Compose command: $compose_cmd"

	cd "$project_root" || return 1

	# Check if Docker daemon is running
	log_verbose "Checking Docker daemon..."
	if ! docker info &>/dev/null; then
		log_error "Docker daemon is not running. Please start Docker."
		return 1
	fi
	log_verbose "Docker daemon is running"

	# Show docker-compose version
	local compose_version
	compose_version=$($compose_cmd version --short 2>/dev/null || echo "unknown")
	log_verbose "Docker Compose version: $compose_version"

	# Check docker-compose.yml exists
	if [ ! -f "docker-compose.yml" ]; then
		log_error "docker-compose.yml not found in: $project_root"
		return 1
	fi

	if [ ! -f "backend/docker-compose.test.yml" ]; then
		log_error "backend/docker-compose.test.yml not found in: $project_root"
		return 1
	fi
	log_verbose "Found docker-compose.yml and backend/docker-compose.test.yml"

	local compose_files=(-f docker-compose.yml -f backend/docker-compose.test.yml)

	# Start services
	local start_cmd="$compose_cmd ${compose_files[*]} up -d"
	log_command "$start_cmd"
	if ! $compose_cmd "${compose_files[@]}" up -d; then
		log_error "Failed to start Docker services"
		log_verbose "Try running: $compose_cmd ${compose_files[*]} logs"
		return 1
	fi

	log_verbose "Services started in detached mode"

	# Show which services are running
	log_verbose "Checking service status..."
	$compose_cmd ps 2>/dev/null | while IFS= read -r line; do
		if [ -n "$line" ]; then
			log_verbose "  $line"
		fi
	done

	log_info "Waiting for services to become healthy..."

	# Wait for PostgreSQL (max 30 seconds)
	log_info "Waiting for PostgreSQL (port 5432)..."
	local count=0
	while [ $count -lt 30 ]; do
		if $compose_cmd ps -q postgres 2>/dev/null | xargs -r docker inspect --format='{{.State.Health.Status}}' 2>/dev/null | grep -q "healthy"; then
			log_success "PostgreSQL is healthy"
			break
		fi
		if [ $((count % 5)) -eq 0 ] && [ $count -gt 0 ]; then
			log_verbose "  Still waiting for PostgreSQL... (${count}s)"
		fi
		sleep 1
		((count++))
	done

	if [ $count -ge 30 ]; then
		log_warn "PostgreSQL health check timeout after 30s"
		log_verbose "Check status with: $compose_cmd ps postgres"
		log_verbose "Check logs with: $compose_cmd logs postgres"
	fi

	# Wait for Redis (max 15 seconds)
	log_info "Waiting for Redis (port 6379)..."
	count=0
	while [ $count -lt 15 ]; do
		if $compose_cmd ps -q redis 2>/dev/null | xargs -r docker inspect --format='{{.State.Status}}' 2>/dev/null | grep -q "running"; then
			log_success "Redis is running"
			break
		fi
		if [ $((count % 5)) -eq 0 ] && [ $count -gt 0 ]; then
			log_verbose "  Still waiting for Redis... (${count}s)"
		fi
		sleep 1
		((count++))
	done

	if [ $count -ge 15 ]; then
		log_warn "Redis health check timeout after 15s"
		log_verbose "Check status with: $compose_cmd ps redis"
		log_verbose "Check logs with: $compose_cmd logs redis"
	fi

	# Show service URLs
	log_success "Docker services started"
	echo -e "  ${C_DIM}PostgreSQL:${C_RESET} localhost:5432"
	echo -e "  ${C_DIM}Redis:${C_RESET}      localhost:6379"
	echo -e "  ${C_DIM}MailHog UI:${C_RESET} http://localhost:8025"

	return 0
}

# docker_down - Stop and remove Docker Compose services
docker_down() {
	local project_root
	local compose_cmd
	project_root="$(get_project_root)"
	compose_cmd="$(get_docker_compose_cmd)"

	log_step "Stopping Docker services..."
	log_verbose "Working directory: $project_root"
	log_verbose "Docker Compose command: $compose_cmd"

	cd "$project_root" || return 1

	# Check if services are actually running
	log_verbose "Checking for running services..."
	local running_services
	running_services=$($compose_cmd ps -q 2>/dev/null | wc -l)

	if [ "$running_services" -eq 0 ]; then
		log_verbose "No services are currently running"
		log_verbose "Cleaning up any orphaned containers..."
	else
		log_verbose "Found $running_services running service(s)"
	fi

	local compose_files=(-f docker-compose.yml -f backend/docker-compose.test.yml)
	local stop_cmd="$compose_cmd ${compose_files[*]} down"
	log_command "$stop_cmd"

	if ! $compose_cmd "${compose_files[@]}" down; then
		log_error "Failed to stop Docker services"
		log_verbose "Try running: docker ps -a"
		log_verbose "Then force remove: docker rm -f \$(docker ps -aq)"
		return 1
	fi

	# Verify cleanup
	log_verbose "Verifying cleanup..."
	local remaining
	remaining=$($compose_cmd ps -q 2>/dev/null | wc -l)
	if [ "$remaining" -gt 0 ]; then
		log_warn "Some services may still be running"
		log_verbose "Check with: $compose_cmd ps"
	else
		log_verbose "All services stopped and containers removed"
	fi

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

	# Check Docker daemon first
	if ! docker info &>/dev/null; then
		echo -e "${C_RED}○${C_RESET} Docker daemon is not running"
		echo ""
		return 1
	fi

	$compose_cmd ps
	echo ""

	# Verbose: show additional details
	if [ "$VERBOSE" -eq 1 ]; then
		echo -e "${C_DIM}────────────────────────────────────────${C_RESET}"
		echo -e "${C_DIM}Detailed Information:${C_RESET}"

		# Show container resource usage
		echo -e "${C_DIM}Container Stats:${C_RESET}"
		docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" 2>/dev/null || echo "  No stats available"

		# Show volume usage
		echo ""
		echo -e "${C_DIM}Volumes:${C_RESET}"
		docker volume ls --format "table {{.Name}}\t{{.Driver}}" 2>/dev/null | head -10

		# Show service URLs
		echo ""
		echo -e "${C_DIM}Service URLs:${C_RESET}"
		echo "  PostgreSQL:  localhost:5432"
		echo "  Redis:       localhost:6379"
		echo "  MailHog UI:  http://localhost:8025"

		# Show helpful commands
		echo ""
		echo -e "${C_DIM}Useful Commands:${C_RESET}"
		echo "  ./scripts/dev.sh logs [service]  - View service logs"
		echo "  $compose_cmd logs [service]     - Direct docker-compose logs"
		echo "  docker stats                     - Live resource usage"
		echo ""
	fi
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

	# Check Docker daemon first
	if ! docker info &>/dev/null; then
		log_error "Docker daemon is not running"
		return 1
	fi

	# Show available services if none specified
	if [ -z "$service" ]; then
		log_verbose "Available services:"
		$compose_cmd ps --services 2>/dev/null | while IFS= read -r svc; do
			log_verbose "  - $svc"
		done
		log_info "Tailing logs for all services (Ctrl+C to exit)..."
		$compose_cmd logs -f
	else
		# Verify service exists
		if ! $compose_cmd ps --services 2>/dev/null | grep -q "^${service}$"; then
			log_error "Service '$service' not found"
			log_verbose "Available services:"
			$compose_cmd ps --services 2>/dev/null | while IFS= read -r svc; do
				log_verbose "  - $svc"
			done
			return 1
		fi
		log_info "Tailing logs for $service (Ctrl+C to exit)..."
		log_verbose "Service: $service"
		$compose_cmd logs -f "$service"
	fi
}

# docker_restart - Restart Docker services
docker_restart() {
	log_step "Restarting Docker services..."
	log_verbose "This will stop all services and start them again"
	docker_down
	sleep 2
	docker_up
}
