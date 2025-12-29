.PHONY: test test-verbose clean

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
