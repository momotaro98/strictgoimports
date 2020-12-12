.PHONY: all
all: test lint

.PHONY: test
test: ## Run all tests
	@go test -v -cover ./...

.PHONY: build
build: ## Run all tests
	@go build -o ./cmd/strictimportsort/strictimportsort ./cmd/strictimportsort

.PHONY: lint
lint: ## Run linter
	go vet `go list ./...`
	@make build
	./cmd/strictimportsort/strictimportsort -local 'github.com/momotaro98/strictimportsort' .

.PHONY: help
help: ## Help command
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
