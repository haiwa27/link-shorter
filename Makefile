SHELL := /bin/bash
COMPOSE := docker compose
SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

.PHONY: help dev build test lint e2e staging-up staging-down prod-up prod-down \
        monitoring-up monitoring-down switch observe rollback clean

help:
	@grep -E '^[a-zA-Z-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

dev: ## Backend lokal starten
	cd app && HEALTHGATE_SLOT=lokal HEALTHGATE_VERSION=$(SHA) go run ./cmd/server

build: ## Backend bauen
	cd app && go build -o bin/server ./cmd/server

test: ## Unit-Tests mit Coverage
	cd app && go test ./... -covermode=count -coverprofile=coverage.out
	cd app && go tool cover -func=coverage.out | tail -1

lint: ## Statische Analyse
	cd app && go vet ./...
	cd app && gofmt -l . | tee /dev/stderr | (! read)

e2e: ## Playwright gegen die Basis-URL aus BASIS_URL
	cd tests/e2e && npx playwright test

staging-up: ## Staging starten
	$(COMPOSE) -f deploy/docker-compose.staging.yml --env-file .env up -d --build

staging-down:
	$(COMPOSE) -f deploy/docker-compose.staging.yml down

prod-up: ## Produktion mit beiden Slots starten
	VERSION_BLUE=$(SHA) VERSION_GREEN=$(SHA) $(COMPOSE) -f deploy/docker-compose.prod.yml --env-file .env up -d --build

prod-down:
	$(COMPOSE) -f deploy/docker-compose.prod.yml down

monitoring-up: ## Prometheus und Grafana starten (Produktion muss laufen)
	$(COMPOSE) -f monitoring/docker-compose.monitoring.yml --env-file .env up -d

monitoring-down:
	$(COMPOSE) -f monitoring/docker-compose.monitoring.yml down

switch: ## Verkehr umschalten, z.B. make switch SLOT=green
	deploy/scripts/switch-slot.sh $(SLOT)

observe: ## Beobachtungsfenster, z.B. make observe SLOT=green
	deploy/scripts/observe.sh $(SLOT)

rollback: ## Auf den vorherigen Slot zurückschalten
	deploy/scripts/rollback.sh

clean:
	rm -rf app/bin app/coverage.out web/dist tests/e2e/test-results
