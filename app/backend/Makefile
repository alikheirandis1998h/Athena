-include .env
export

MIGRATIONS_DIR=migrations

.PHONY: run build migrate-up migrate-down sqlc-generate swag-init docker-up docker-down test

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

sqlc-generate:
	sqlc generate

swag-init:
	swag init -g cmd/api/main.go -o docs

docker-up:
	docker compose up -d

docker-down:
	docker compose down

test:
	go test ./...
