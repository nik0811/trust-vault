.PHONY: build run test clean migrate docker-up docker-down

# Build
build:
	go build -o bin/server ./cmd/server
	go build -o bin/migrate ./cmd/migrate

# Run
run: build
	./bin/server --mode=gateway --port=8080

run-worker: build
	./bin/server --mode=worker

# Test
test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Database
migrate-up:
	./bin/migrate --direction=up

migrate-down:
	./bin/migrate --direction=down

# Docker
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker build -t trustvault:latest .
	docker build -t trustvault-docservice:latest -f docservice/Dockerfile .

# Development
dev:
	go run ./cmd/server --mode=gateway --port=8080

dev-worker:
	go run ./cmd/server --mode=worker

# Lint
lint:
	golangci-lint run

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Generate
generate:
	go generate ./...

# Dependencies
deps:
	go mod download
	go mod tidy

# All
all: deps build test
