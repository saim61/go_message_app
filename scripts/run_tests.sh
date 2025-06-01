#!/bin/bash

# Test runner script for Go Message App
# This script runs all tests with coverage reporting

set -e

echo "🧪 Running Go Message App Test Suite"
echo "===================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

print_status "Go version: $(go version)"

# Create coverage directory if it doesn't exist
mkdir -p coverage

# Set test environment variables
export JWT_SECRET="test-secret-key-for-testing"
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_NAME="test_db"
export DB_USER="test_user"
export DB_PASSWORD="test_password"

print_status "Running unit tests..."

# Run tests with coverage
go test -v -race -coverprofile=coverage/coverage.out ./... 2>&1 | tee coverage/test_output.log

# Check if tests passed
if [ ${PIPESTATUS[0]} -eq 0 ]; then
    print_success "All tests passed!"
else
    print_error "Some tests failed. Check the output above."
    exit 1
fi

# Generate coverage report
print_status "Generating coverage report..."
go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Display coverage summary
print_status "Coverage summary:"
go tool cover -func=coverage/coverage.out | tail -1

# Check coverage threshold (optional)
COVERAGE_THRESHOLD=70
COVERAGE=$(go tool cover -func=coverage/coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')

if (( $(echo "$COVERAGE >= $COVERAGE_THRESHOLD" | bc -l) )); then
    print_success "Coverage ($COVERAGE%) meets threshold ($COVERAGE_THRESHOLD%)"
else
    print_warning "Coverage ($COVERAGE%) is below threshold ($COVERAGE_THRESHOLD%)"
fi

print_status "Coverage report generated: coverage/coverage.html"

# Run specific test categories
echo ""
print_status "Running test categories..."

echo ""
print_status "🔐 Authentication Tests"
go test -v ./internal/auth/... -run "Test.*"

echo ""
print_status "🗄️  Storage Tests"
go test -v ./internal/storage/... -run "Test.*"

echo ""
print_status "🌐 Gateway Tests"
go test -v ./internal/gateway/... -run "Test.*"

echo ""
print_status "🛣️  Route Tests"
go test -v ./routes/... -run "Test.*"

echo ""
print_status "📡 HTTP Utilities Tests"
go test -v ./internal/httpx/... -run "Test.*"

# Run integration tests separately
echo ""
print_status "🔗 Integration Tests"
go test -v ./internal/gateway/... -run "TestIntegration.*"

# Run benchmarks (optional)
if [ "$1" = "--bench" ]; then
    echo ""
    print_status "🏃 Running benchmarks..."
    go test -bench=. -benchmem ./...
fi

# Check for race conditions
echo ""
print_status "🏁 Checking for race conditions..."
go test -race ./...

print_success "Test suite completed successfully!"
print_status "View detailed coverage report: open coverage/coverage.html" 