.PHONY: tools
tools: ## Run tools (vet, gofmt, goimports, tidy, etc.)
	@go version
	gofmt -w .
	goimports -w .
	go mod tidy
	go vet ./...

.PHONY: tools.update
tools.update: ## Update or install tools
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/vektra/mockery/v2@v2.40.1

.PHONY: deps.update
deps.update: ## Update dependencies versions
	go get -u all
	go mod tidy

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: down
down: ## Down dest database
	docker compose -f docker compose.yml down

.PHONY: up
up: ## Up dest database
	COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_BUILDKIT=1 docker compose -f docker-compose.yml up --build --force-recreate

.PHONY: up.d
up.d: ## Up dest database (detached)
	COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_BUILDKIT=1 docker compose -f docker-compose.yml up -d --build --force-recreate

.PHONY: help
help: ## List all make targets with description
	@grep -h -E '^[.a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: list
list: ## List all make targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort
