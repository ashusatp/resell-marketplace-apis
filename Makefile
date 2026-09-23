.PHONY: build run clean

build:
	@go build -o bin/api ./cmd/api

run: build
	@./bin/api

clean:
	@rm -rf bin

migrate-up:
	@go run ./cmd/migrate up

migrate-down:
	@go run ./cmd/migrate down

migrate-reset:
	@go run ./cmd/migrate reset

migrate-create:
	@migrate create -ext sql -dir migrations -seq $(name)