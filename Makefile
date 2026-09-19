.PHONY: build run clean

build:
	@go build -o bin/api ./cmd/api

run: build
	@./bin/api

clean:
	@rm -rf bin
