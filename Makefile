.PHONY: test lint run-auth tidy

test:
	go test ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

run-auth:
	go run ./cmd/auth
