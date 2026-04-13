# A0Hero — Makefile

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-12s\033[0m %s\n", $$1, $$2}'

## ========== DEVELOPMENT ==========

install: ## Install dependencies
	go mod download
	go mod tidy

tidy: ## Tidy go modules
	go mod tidy

lint: ## Run linters
	golangci-lint run ./...
	@go fmt ./...

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

## ========== BUILDING ==========

build: ## Build the a0hero binary
	go build -o bin/a0hero ./cmd/a0hero

clean: ## Remove build artifacts
	rm -rf bin/

## ========== TESTING ==========

test: ## Run all tests (verbose)
	go test ./... -v

test-cover: ## Run tests with coverage report
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-watch: ## Run tests on file changes (requires entr or similar)
	find . -name "*.go" | entr -r go test ./... -v

## ========== CODE GENERATION ==========

generate: ## Generate types and mocks from OpenAPI spec
	@echo "Fetching Auth0 OpenAPI spec..."
	curl -s https://auth0.com/docs/api/management/v2/openapi.yaml -o tests/mocks/auth0/openapi.yaml
	@echo "Generating Go types..."
	oapi-codegen -generate types -package auth0 -o models/generated.go tests/mocks/auth0/openapi.yaml
	@echo "Generating mock server..."
	oapi-codegen -generate chi-server -package mockauth0 -o tests/mocks/auth0/server.go tests/mocks/auth0/openapi.yaml
	@echo "Done."

## ========== RUNNING ==========

run: ## Run a0hero (from config/dev.yaml by default)
	a0hero run

run-dev: ## Run with dev tenant
	AUTH0_TENANT=dev a0hero run

run-prod: ## Run with prod tenant
	AUTH0_TENANT=prod a0hero run

configure: ## Launch the configure wizard
	a0hero configure

## ========== CI/CD ==========

ci-test: ## Run tests as CI would (no TTY, no cached modules)
	go test ./... -v -count=1

ci-lint: ## Run linters as CI would
	golangci-lint run ./...

## ========== MISC ==========

review-arch: ## Review client package architecture
	@echo "Checking client/ package structure..."
	@ls -la client/
	@echo "Checking module imports for TUI violations..."
	@grep -r "tui" modules/ 2>/dev/null && echo "VIOLATION: modules/ must not import tui/" || echo "OK: modules/ does not import tui/"

smoke: ## Quick smoke test — build + run --help
	go build -o bin/a0hero ./cmd/a0hero
	./bin/a0hero --help