.PHONY: build run dev test clean db-up db-down migrate seed deploy-local ui theme

BINARY=squilla
CMD=./cmd/squilla

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

ui:
	cd admin-ui && npm run build --silent
	docker cp admin-ui/dist/. squilla-app-1:/app/admin-ui/dist/
	@echo "==> UI hot-copied into container"

THEME ?= squilla

theme:
	@echo "==> Hot-copying theme '$(THEME)' into container..."
	docker compose cp themes/$(THEME)/. app:/app/themes/$(THEME)/
	docker compose restart app
	@echo "==> Theme '$(THEME)' hot-deployed (use THEME=<slug> to target a different theme)"

deploy-local:
	@echo "==> Building admin UI..."
	cd admin-ui && npm ci --silent && node scripts/generate-icon-shim.cjs && npm run build
	@echo "==> Building extension admin UIs..."
	@for dir in extensions/*/admin-ui; do \
		[ -f "$$dir/package.json" ] || continue; \
		slug=$$(basename $$(dirname $$dir)); \
		echo "    -> $$slug"; \
		cd $$dir && npm install --silent && npm run build && cd ../../..; \
	done
	@echo "==> Building Go binary (linux/arm64)..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o squilla ./cmd/squilla
	@echo "==> Building extension plugins (linux/arm64)..."
	@for dir in extensions/*/cmd/plugin; do \
		[ -f "$$dir/main.go" ] || continue; \
		slug=$$(echo "$$dir" | cut -d/ -f2); \
		echo "    -> $$slug"; \
		mkdir -p extensions/$$slug/bin; \
		GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o extensions/$$slug/bin/$$slug ./$$dir/; \
	done
	@echo "==> Building Docker image..."
	docker build -f Dockerfile.local -t squilla:local .
	@echo "==> Starting containers..."
	docker compose up -d --no-build
	@echo "==> Done. App running at http://localhost:8099"
