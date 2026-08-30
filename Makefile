SUBMODULES=test

.PHONY: godoc
godoc: ## Install and run godoc
	go install golang.org/x/tools/cmd/godoc@latest
	godoc -http=:6060

.PHONY: tools
tools: ## Run tools (vet, gofmt, goimports, tidy, etc.)
	@go version
	@(for sub in ${SUBMODULES} ; do \
		pushd "$$sub" && gofmt -w . && goimports -w . && go mod tidy && GIT_TRACE=1 GIT_CURL_VERBOSE=1 go mod download -x && popd; \
	done)
	@go mod tidy
	@GIT_TRACE=1 GIT_CURL_VERBOSE=1 go mod download -x

.PHONY: tools.update
tools.update: ## Update or install tools
	@echo "available versions:"
	@go list -m -versions github.com/golangci/golangci-lint/v2
	@go list -m -versions golang.org/x/tools/cmd/goimports
	@echo "install:"
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2

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
	@golangci-lint run ./...

.PHONY: tags.list
tags.list: ## List all exists 'git' tags
	@(git tag | sort -rt "." -k1,1n -k2,2n -k3,3n | tail -r)

.PHONY: git.log
git.log: ## Print formatted git log from "start commit" to HEAD (args: c - start commit)
	@(val=$$(echo $(c)| tr -d ' ') && \
	git log --pretty=format:"* %H %s" $c..HEAD)

.PHONY: git.del.local.br
git.del.local.br: ## Delete local branches by pattern (args: b - branch pattern).
	@(val=$$(echo $(b)| tr -d ' ') && \
  	echo "delete branches by pattern: $$val" && \
	git for-each-ref --format='%(refname:short)' refs/heads \
      | grep "^$$val" \
      | xargs git branch -D)

.PHONY: git.del.local.dependabot.branches
git.del.local.dependabot.branches: ## Delete local "dependabot" branches.
	$(MAKE) git.del.local.br b=dependabot/

.PHONY: help
help: ## List all 'make' targets with description
	@grep -h -E '^[.a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: list
list: ## List all 'make' targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort
