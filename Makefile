.PHONT: tools
tools: ## Run tools (vet, gofmt, goimports, tidy, etc.)
	@go version
	gofmt -w .
	goimports -w .
	go mod tidy
	go vet ./...

.PHONT: tools.update
tools.update: ## Update or install tools
	go install golang.org/x/tools/cmd/goimports@latest

.PHONT: deps.update
deps.update: ## Update dependencies versions
	go get -u all
	go mod tidy

.PHONT: test
test: ## Run tests with coverage
	go test ./... -cover

.PHONT: test.cover.all
test.cover.all: ## Run tests with coverage (show all coverage)
	go test -v ./... -cover -coverprofile cover.out  && go tool cover -func cover.out

.PHONY: lint
lint: ## Run `golangci-lint`
	@go version
	@golangci-lint --version
	@golangci-lint run .

.PHONY: help
help: ## List all make targets with description
	@grep -h -E '^[.a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: list
list: ## List all make targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort
