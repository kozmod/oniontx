.PHONY: tools
tools: ## Run tools (vet, gofmt, goimports, tidy, etc.)
	@go version
	gofmt -w .
	goimports -w .
	go mod tidy
	go vet ./...

.PHONY: tools.install
tools.install: ## Install tool for tests
	go install go.uber.org/mock/mockgen@latest
	go install github.com/gojuno/minimock/v3/cmd/minimock@latest
	go install github.com/vektra/mockery/v3@v3.5.1

.PHONY: deps.update
deps.update: ## Update dependencies versions
	go get -u all
	go mod tidy

.PHONY: generate
generate: ## Run `go generate`
	@go generate ./...

.PHONY: help
help: ## List all make targets with description
	@grep -h -E '^[.a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: list
list: ## List all make targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort
