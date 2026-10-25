BINARY := tmc
PKG    := ./cmd/tmc

.DEFAULT_GOAL := help

.PHONY: help build install run test test-race cover vet fmt fmt-check tidy check clean

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

build: ## Build ./tmc
	go build -o $(BINARY) $(PKG)

install: ## Install tmc into your Go bin directory
	go install $(PKG)

run: ## Run tmc with ARGS, e.g. make run ARGS="asana --help"
	go run $(PKG) $(ARGS)

test: ## Run unit tests
	go test ./...

test-race: ## Run unit tests with the race detector
	go test -race ./...

cover: ## Run tests and open an HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out

vet: ## Run go vet
	go vet ./...

fmt: ## Format all Go files
	gofmt -w .

fmt-check: ## Fail if any Go files need formatting
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "Needs gofmt:"; echo "$$unformatted"; exit 1; fi

tidy: ## Tidy go.mod and go.sum
	go mod tidy

check: fmt-check vet test ## Run formatting, vet and tests (what CI should run)

clean: ## Remove build and coverage output
	rm -f $(BINARY) coverage.out
