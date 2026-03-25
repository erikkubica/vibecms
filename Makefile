.PHONY: build run dev test clean db-up db-down migrate seed

BINARY=vibecms
CMD=./cmd/vibecms

build:
	go build -o bin/$(BINARY) $(CMD)

run: build
	./bin/$(BINARY)

dev:
	go run $(CMD)

test:
	go test ./... -v -race

clean:
	rm -rf bin/

db-up:
	docker compose up -d db

db-down:
	docker compose down

migrate:
	go run $(CMD) migrate

seed:
	go run $(CMD) seed

lint:
	golangci-lint run ./...
