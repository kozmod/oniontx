TAG_REGEXP=^[v][0-9]+[.][0-9]+[.][0-9]+([-]{0}|[-]{1}[0-9a-zA-Z]+[.]?[0-9a-zA-Z]+)+$$
SUBMODULES=test

.PHONY: godoc
godoc: ## Install and run godoc
	go install golang.org/x/tools/cmd/godoc@latest
	godoc -http=:6060

.PHONY: tools
tools: ## Run tools (vet, gofmt, goimports, tidy, etc.)
	@go version
	@(for sub in ${SUBMODULES} ; do \
		pushd "$$sub" && gofmt -w . && goimports -w . && go mod tidy && go mod download && popd; \
	done)
	@go mod tidy
	@go mod download

.PHONY: tools.update
tools.update: ## Update or install tools
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: deps.update
deps.update: ## Update dependencies versions (root and sub modules)
	@GOTOOLCHAIN=local go get -u all
	@(for sub in ${SUBMODULES} ; do \
		pushd "$$sub" && go get -u all && go mod tidy && go mod download && popd; \
	done)
	@go mod tidy
	@go mod download
	@go work sync

.PHONY: go.sync
go.sync: ## Sync modules
	@go work sync

.PHONY: test
test: ## Run tests with coverage
	@go test ./... -cover

.PHONY: test.cover.all
test.cover.all: ## Run tests with coverage (show all coverage)
	@go test -v ./... -cover -coverprofile cover.out  && go tool cover -func cover.out

.PHONY: lint
lint: ## Run `golangci-lint`
	@go version
	@golangci-lint --version
	@golangci-lint run .

.PHONY: tags.add
tags.add: ## Add root module and submodules tags (args: t=<v*.*.*-*.*>)(git)
	@(val=$$(echo $(t)| tr -d ' ') && \
	branch=$$(git rev-parse --abbrev-ref HEAD) && \
	if [[ ! $$val =~ ${TAG_REGEXP} ]] ; then echo "not semantic version tag [$$val]" && exit 2; fi && \
	git tag "$$val" && echo "add root module's tag [$$val] on branch [$$branch]"

.PHONY: tags.del
tags.del: ## Delete root module and submodules tags (args: t=<v*.*.*-*.*>)(git)
	@(val=$$(echo $(t)| tr -d ' ') && \
	if [[ ! $$val =~ ${TAG_REGEXP} ]] ; then echo "not semantic version tag [$$val]" && exit 2; fi && \
	git tag --delete "$$val"

.PHONY: tags.list
tags.list: ## List all exists tags (git)
	@(git tag | sort -rt "." -k1,1n -k2,2n -k3,3n | tail -r)

.PHONY: help
help: ## List all make targets with description
	@grep -h -E '^[.a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: list
list: ## List all make targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort
