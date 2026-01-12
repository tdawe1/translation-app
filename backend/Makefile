.PHONY: test test-verbose test-coverage clean test-db-up test-db-down test-db-logs test-setup test-with-setup test-db-create

# Run tests with default settings
test:
	JWT_SECRET=test-secret-for-testing-only-32-chars-min go test ./tests/... -count=1

# Run tests with verbose output
test-verbose:
	JWT_SECRET=test-secret-for-testing-only-32-chars-min go test ./tests/... -v -count=1

# Run tests with coverage
test-coverage:
	JWT_SECRET=test-secret-for-testing-only-32-chars-min go test ./tests/... -coverprofile=coverage.out -count=1
	go tool cover -html=coverage.out -o coverage.html

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html

# Test Infrastructure
.PHONY: test-db-up test-db-down test-db-logs test-with-setup test-db-create

# Start test database (runs in background)
test-db-up:
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker exec gengowatcher-postgres-test pg_isready -U gengo -p 5433 > /dev/null 2>&1; do \
		echo "Waiting for postgres..."; \
		sleep 1; \
	done
	@echo "Test database is ready on port 5433"

# Stop test database
test-db-down:
	docker-compose -f docker-compose.test.yml down

# View test database logs
test-db-logs:
	docker-compose -f docker-compose.test.yml logs -f

# Run tests with database setup
test-with-setup: test-db-up
	@$(MAKE) test
	@$(MAKE) test-db-down

# Create test database (for manual testing)
test-db-create:
	docker-compose -f docker-compose.test.yml up -d
	@sleep 3
	docker exec gengowatcher-postgres-test psql -U gengo -p 5433 -c "SELECT 1"
