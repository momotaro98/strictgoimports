.PHONY: all
all: test lint

.PHONY: test
test: ## Run all tests
	@go test -v -cover ./...

.PHONY: build
build: ## Run all tests
	@go build -o ./cmd/strictgoimports/strictgoimports ./cmd/strictgoimports

.PHONY: lint
lint: ## Run linter
	go vet `go list ./...`
	@make build
	./cmd/strictgoimports/strictgoimports -local 'github.com/momotaro98/strictgoimports' .

.PHONY: help
help: ## Help command
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
