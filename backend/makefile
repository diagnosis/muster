PHONY: lint run migrate-up migrate-down migrate-status migrate-create

-include .env
export

# linting
lint:
	golangci-lint run ./...

# running app
run:
	go run cmd/muster/main.go

# run migrations
migrate-up:
	@goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	@goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status:
	@goose -dir migrations postgres "$(DATABASE_URL)" status

migrate-create:
	@goose -dir migrations create $(name) sql