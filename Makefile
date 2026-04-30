# Rate Guard Makefile

.PHONY: all lint test build proto run
all: lint test build proto

lint:
	@echo "Running linter..."
	@golint -set_exit_status ./... 2>/dev/null || true

test:
	@echo "Running tests..."
	@go test -race ./...

build:
	@echo "Building rate-guard..."
	@go build -o bin/rate-guard ./cmd/rate-guard

proto:
	@echo "Generating protobuf code..."
	@if not exist pkg\pb mkdir pkg\pb
	@protoc --proto_path=proto \
		--go_out=pkg/pb --go_opt=paths=source_relative \
		--go-grpc_out=pkg/pb --go-grpc_opt=paths=source_relative \
		ratelimit/v1/ratelimit.proto

run:
	@echo "Running rate-guard..."
	@go run cmd/server/main.go

docker-up:
	@docker-compose up

docker-down:
	@docker-compose down