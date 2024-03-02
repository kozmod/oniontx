TAG_TEMPLATE:= ^[v][0-9]+[.][0-9]+[.][0-9]([-]{0}|[-]{1}[0-9a-zA-Z]+[.]?[0-9a-zA-Z]+)+$$

PHONT: tools
tools: ## Run tools (vet, gofmt, goimports, tidy, etc.)
	@go version
	gofmt -w .
	goimports -w .
	go mod tidy
	go vet ./...

.PHONT: tools.update
tools.update: ## Update or install tools
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONT: deps.update
deps.update: ## Update dependencies versions (root and sub modules)
	@go get -u all
	@go mod tidy
	@for d in */ ; do pushd "$$d" && go get -u all && go mod tidy && popd; done

.PHONT: go.sync
go.sync: ## Sync modules
	@go work sync

.PHONT: test
test: ## Run tests with coverage
	@go test ./... -cover

.PHONT: test.cover.all
test.cover.all: ## Run tests with coverage (show all coverage)
	@go test -v ./... -cover -coverprofile cover.out  && go tool cover -func cover.out

.PHONY: lint
lint: ## Run `golangci-lint`
	@go version
	@golangci-lint --version
	@golangci-lint run .

.PHONT: tags.add
tags.add: ## Set root module and submodules tags (git)
	@(val=$$(echo $(t)| tr -d ' ') && \
	if [[ ! $$val =~ ${TAG_TEMPLATE} ]] ; then echo "not semantic version tag [$$val]" && exit 2; fi && \
	git tag "$$val" && echo "set root module's tag [$$val]" && \
	for d in */ ; do git tag "$$d$$val" && echo "set submodule's tag [$$d$$val]"; done)

.PHONT: tags.del
tags.del: ## Delete root module and submodules tags (git)
	@(val=$$(echo $(t)| tr -d ' ') && \
	if [[ ! $$val =~ ${TAG_TEMPLATE} ]] ; then echo "not semantic version tag [$$val]" && exit 2; fi && \
	git tag --delete "$$val" && echo "delete root module's tag [$$val]" && \
	for d in */ ; do git tag --delete "$$d$$val" && echo "delete submodule's tag [$$d$$val]"; done)

.PHONT: tags.list
tags.list: ## List all exists tags (git)
	@(git for-each-ref refs/tags --sort=-taggerdate --format='%(refname)')

.PHONY: help
help: ## List all make targets with description
	@grep -h -E '^[.a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: list
list: ## List all make targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort
