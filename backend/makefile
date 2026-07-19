.PHONY: fmt lint build test check run psql migrate-up migrate-down migrate-status migrate-create

-include .env
export

# format all Go code
fmt:
	gofmt -w ./cmd ./internal

# vet + lint
lint:
	golangci-lint run ./...

# build everything
build:
	go build ./...

# run all tests
test:
	go test ./...

# the full pre-commit gate
check: fmt lint build test

# running app
run:
	go run cmd/muster/main.go

# open a psql shell against the configured database
psql:
	@psql "$(DATABASE_URL)"

# run migrations
migrate-up:
	@goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	@goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status:
	@goose -dir migrations postgres "$(DATABASE_URL)" status

migrate-create:
	@goose -dir migrations create $(name) sql