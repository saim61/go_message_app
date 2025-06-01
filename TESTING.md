# Testing Guide for Go Message App

This document provides comprehensive information about the testing strategy, test structure, and how to run tests for the Go Message App.

## 📋 Table of Contents

- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Test Categories](#test-categories)
- [Coverage Reports](#coverage-reports)
- [Writing New Tests](#writing-new-tests)
- [Mocking Strategy](#mocking-strategy)
- [CI/CD Integration](#cicd-integration)

## 🏗️ Test Structure

The project follows Go testing conventions with comprehensive test coverage across all components:

```
├── internal/
│   ├── auth/
│   │   ├── password_test.go      # Password hashing/verification tests
│   │   └── token_test.go         # JWT token generation/parsing tests
│   ├── gateway/
│   │   ├── hub_test.go           # WebSocket hub functionality tests
│   │   ├── websocket_test.go     # WebSocket handler tests
│   │   ├── kafka_test.go         # Kafka consumer tests
│   │   └── integration_test.go   # End-to-end integration tests
│   ├── httpx/
│   │   └── response_test.go      # HTTP response utility tests
│   └── storage/
│       └── postgres/
│           ├── user_repo_test.go     # User repository tests
│           └── message_repo_test.go  # Message repository tests
├── routes/
│   └── auth_test.go              # Authentication route tests
└── scripts/
    └── run_tests.sh              # Test runner script
```

## 🚀 Running Tests

### Quick Start

Run all tests with the test runner script:

```bash
./scripts/run_tests.sh
```

### Manual Test Commands

#### Run All Tests
```bash
go test ./...
```

#### Run Tests with Coverage
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

#### Run Tests with Race Detection
```bash
go test -race ./...
```

#### Run Specific Test Package
```bash
go test ./internal/auth/...
go test ./internal/gateway/...
go test ./internal/storage/...
go test ./routes/...
```

#### Run Specific Test Function
```bash
go test -run TestNewToken ./internal/auth/...
go test -run TestIntegration.* ./internal/gateway/...
```

#### Run Tests with Verbose Output
```bash
go test -v ./...
```

#### Run Benchmarks
```bash
./scripts/run_tests.sh --bench
```

## 📊 Test Categories

### 1. Unit Tests

**Authentication (`internal/auth/`)**
- `TestNewToken`: Token generation with various scenarios
- `TestParseToken`: Token parsing and validation
- `TestHashPassword`: Password hashing functionality
- `TestCheckPassword`: Password verification

**Gateway Hub (`internal/gateway/hub_test.go`)**
- `TestNewHub`: Hub initialization
- `TestHubRegisterClient`: Client registration
- `TestHubUnregisterClient`: Client unregistration
- `TestHubBroadcast`: Message broadcasting
- `TestGetRoomUserCount`: Room user counting

**WebSocket Handler (`internal/gateway/websocket_test.go`)**
- `TestParseToken`: JWT token parsing in WebSocket context
- `TestWSHandler_Authentication`: WebSocket authentication
- `TestClientReadPump`: Message reading functionality
- `TestClientWritePump`: Message writing functionality

**Kafka Consumer (`internal/gateway/kafka_test.go`)**
- `TestConsumeMessages_*`: Various Kafka consumption scenarios
- Error handling and message processing tests

**Storage (`internal/storage/postgres/`)**
- `TestUserRepo_*`: User repository operations
- `TestMessageRepo_*`: Message repository operations
- Database interaction and error handling tests

**HTTP Utilities (`internal/httpx/`)**
- `TestOK`: Success response formatting
- `TestFail`: Error response formatting

**Routes (`routes/`)**
- `TestAuthRoutes_Register`: User registration endpoint
- `TestAuthRoutes_Login`: User login endpoint
- HTTP request/response testing

### 2. Integration Tests

**End-to-End (`internal/gateway/integration_test.go`)**
- `TestIntegration_WebSocketConnection`: Full WebSocket flow
- `TestIntegration_MultipleClients`: Multi-client scenarios
- `TestIntegration_RoomSwitching`: Room switching functionality
- `TestIntegration_ClientDisconnection`: Disconnection handling
- `TestIntegration_ConcurrentConnections`: Concurrent client handling

## 📈 Coverage Reports

### Viewing Coverage

After running tests with coverage:

```bash
# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# View coverage summary
go tool cover -func=coverage.out

# Open coverage report in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Coverage Targets

- **Overall Coverage**: Target 70%+
- **Critical Components**: Target 90%+
  - Authentication (`internal/auth/`)
  - WebSocket Hub (`internal/gateway/hub.go`)
  - Storage Repositories (`internal/storage/`)

## ✍️ Writing New Tests

### Test File Naming

Follow Go conventions:
- Test files: `*_test.go`
- Test functions: `TestFunctionName`
- Benchmark functions: `BenchmarkFunctionName`

### Test Structure

```go
func TestFunctionName(t *testing.T) {
    // Arrange
    // Set up test data and dependencies
    
    // Act
    // Execute the function being tested
    
    // Assert
    // Verify the results
    assert.Equal(t, expected, actual)
}
```

### Table-Driven Tests

For multiple test scenarios:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "expected", false},
        {"invalid input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

## 🎭 Mocking Strategy

### Database Mocking

Using `sqlmock` for database interactions:

```go
func TestDatabaseFunction(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()
    
    mock.ExpectQuery("SELECT").WillReturnRows(
        sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "test"),
    )
    
    // Test your function
}
```

### WebSocket Mocking

Using `testify/mock` for WebSocket components:

```go
type MockHub struct {
    mock.Mock
}

func (m *MockHub) Broadcast(message WireMessage) {
    m.Called(message)
}
```

### Kafka Mocking

Custom mocks for Kafka consumers and producers:

```go
type MockConsumer struct {
    mock.Mock
}

func (m *MockConsumer) ConsumePartition(topic string, partition int32, offset int64) (sarama.PartitionConsumer, error) {
    args := m.Called(topic, partition, offset)
    return args.Get(0).(sarama.PartitionConsumer), args.Error(1)
}
```

## 🔄 CI/CD Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - name: Run tests
        run: ./scripts/run_tests.sh
      - name: Upload coverage
        uses: codecov/codecov-action@v1
        with:
          file: ./coverage/coverage.out
```

### Pre-commit Hooks

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: local
    hooks:
      - id: go-test
        name: go test
        entry: ./scripts/run_tests.sh
        language: system
        pass_filenames: false
```

## 🛠️ Test Environment Setup

### Environment Variables

Set these for testing:

```bash
export JWT_SECRET="test-secret-key"
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_NAME="test_db"
export DB_USER="test_user"
export DB_PASSWORD="test_password"
```

### Test Database

For integration tests requiring a real database:

```bash
# Start test database with Docker
docker run --name test-postgres \
  -e POSTGRES_DB=test_db \
  -e POSTGRES_USER=test_user \
  -e POSTGRES_PASSWORD=test_password \
  -p 5432:5432 -d postgres:13

# Run migrations
migrate -path migrations -database "postgres://test_user:test_password@localhost:5432/test_db?sslmode=disable" up
```

## 📝 Best Practices

1. **Test Independence**: Each test should be independent and not rely on other tests
2. **Clear Naming**: Use descriptive test names that explain what is being tested
3. **Arrange-Act-Assert**: Structure tests clearly with setup, execution, and verification
4. **Edge Cases**: Test both happy path and error scenarios
5. **Mocking**: Mock external dependencies to isolate units under test
6. **Coverage**: Aim for high coverage but focus on meaningful tests
7. **Performance**: Include benchmark tests for performance-critical code
8. **Documentation**: Document complex test scenarios and setup requirements

## 🐛 Debugging Tests

### Verbose Output
```bash
go test -v ./...
```

### Run Single Test
```bash
go test -run TestSpecificFunction ./package/...
```

### Debug with Delve
```bash
dlv test ./package/... -- -test.run TestSpecificFunction
```

### Test Timeout
```bash
go test -timeout 30s ./...
```

## 📚 Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Go Test Coverage](https://blog.golang.org/cover)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests) 