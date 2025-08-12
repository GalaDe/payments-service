# ---- Config ----
APP_NAME := payments-service
API_CMD  := cmd/api/main.go
WORKER_CMD := cmd/worker/main.go

# If your docker compose file has another name, override with: make dc="docker compose -f path.yml" <target>
dc ?= docker compose

# ---- Help ----
.PHONY: help
help: ## Show this help
	@echo 'Usage: make <target>'
	@echo ''
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | sed 's/:.*##/: /' | column -t -s': '

# ---- Go basics ----
.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: fmt
fmt: ## go fmt
	go fmt ./...

.PHONY: vet
vet: ## go vet
	go vet ./...

.PHONY: test
test: ## run unit tests
	go test ./... -count=1

.PHONY: check
check: tidy fmt vet test ## run tidy, fmt, vet, test

# ---- sqlc ----
.PHONY: sqlc
sqlc: ## generate sqlc code
	sqlc generate

# ---- Run services ----
.PHONY: run-api
run-api: ## run HTTP API locally
	go run $(API_CMD)

.PHONY: run-worker
run-worker: ## run Temporal worker locally
	go run $(WORKER_CMD)

# ---- Docker / infra (optional) ----
.PHONY: up
up: ## start docker-compose stack (db, temporal, etc.)
	$(dc) up -d

.PHONY: down
down: ## stop docker-compose stack
	$(dc) down

.PHONY: logs
logs: ## tail compose logs
	$(dc) logs -f

# ---- Temporal helpers (optional) ----
.PHONY: temporal-ns
temporal-ns: ## list Temporal namespaces (requires temporal CLI)
	temporal --address localhost:7233 namespace list

# ---- Cleaning ----
.PHONY: clean
clean: ## no-op clean placeholder (add build artifacts if any)
	@echo "Nothing to clean."
