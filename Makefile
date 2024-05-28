TAG_REGEXP=^[v][0-9]+[.][0-9]+[.][0-9]+([-]{0}|[-]{1}[0-9a-zA-Z]+[.]?[0-9a-zA-Z]+)+$$
TAG_SUBMODULES=stdlib pgx sqlx gorm
SUBMODULES=${TAG_SUBMODULES} test

.PHONT: tools
tools: ## Run tools (vet, gofmt, goimports, tidy, etc.)
	@go version
	@(for sub in ${SUBMODULES} ; do \
		pushd "$$sub" && gofmt -w . && goimports -w . && go mod tidy && go mod download && popd; \
	done)
	@go mod tidy
	@go mod download

.PHONT: tools.update
tools.update: ## Update or install tools
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONT: deps.update
deps.update: ## Update dependencies versions (root and sub modules)
	@GOTOOLCHAIN=local go get -u all
	@(for sub in ${SUBMODULES} ; do \
		pushd "$$sub" && go get -u all && go mod tidy && go mod download && popd; \
	done)
	@go mod tidy
	@go mod download
	@go work sync

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
tags.add: ## Add root module and submodules tags (args: t=<v*.*.*-*.*>)(git)
	@(val=$$(echo $(t)| tr -d ' ') && \
	branch=$$(git rev-parse --abbrev-ref HEAD) && \
	if [[ ! $$val =~ ${TAG_REGEXP} ]] ; then echo "not semantic version tag [$$val]" && exit 2; fi && \
	git tag "$$val" && echo "add root module's tag [$$val] on branch [$$branch]" && \
	for sub in ${TAG_SUBMODULES} ; do \
		git tag "$$sub/$$val" && echo "add submodule's tag [$$sub/$$val] on branch [$$branch]"; \
	done)

.PHONT: tags.del
tags.del: ## Delete root module and submodules tags (args: t=<v*.*.*-*.*>)(git)
	@(val=$$(echo $(t)| tr -d ' ') && \
	if [[ ! $$val =~ ${TAG_REGEXP} ]] ; then echo "not semantic version tag [$$val]" && exit 2; fi && \
	git tag --delete "$$val" && \
	for sub in ${TAG_SUBMODULES} ; do \
    	git tag --delete "$$sub/$$val"; \
    done)

.PHONT: tags.add.last
tags.add.last: ## Add tags for submodules based on last tag (matches v*.*.*) which point to last commit (git)
	@( \
	git fetch --tags --force && \
	last_tag=$$(git tag | grep -E '^[v][0-9]+[.][0-9]+[.][0-9]+$$' | sort -t "." -k1,1n -k2,2n -k3,3n | tail -1) &&\
	last_tag_hash=$$(git rev-list -n 1 "$$last_tag") && \
	last_commit_hash=$$(git rev-parse HEAD) && \
	branch=$$(git rev-parse --abbrev-ref HEAD) && \
	if [[ $$last_tag_hash == $$last_commit_hash ]]; then \
		for sub in ${TAG_SUBMODULES} ; do \
			git tag "$$sub/$$last_tag" && echo "add submodule's tag [$$sub/$$last_tag] on branch [$$branch]"; \
		done; \
	else \
		echo "last tag [$$last_tag] hash [$$last_tag_hash] and last commit hash [$$last_commit_hash] are different" ; \
	fi \
	)

.PHONT: tags.list
tags.list: ## List all exists tags (git)
	@(git tag | sort -rt "." -k1,1n -k2,2n -k3,3n | tail -r)

.PHONT: change.sub.lib.version
change.sub.lib.version: ## Change submodules `oniontx` deps version (args: t=<v*.*.*-*.*>)
	@(val=$$(echo $(t)| tr -d ' ') && \
	if [[ ! $$val =~ ${TAG_REGEXP} ]]; then \
		echo "version (tag) of the oniontx is not semantic version tag [$$val]" && exit 2; \
	else \
		for sub in ${TAG_SUBMODULES} ; do \
			pushd $$sub && \
			echo "change go mod version(tag) of the oniontx to [$$val]" && \
			sed -i '' "s/github.com\/kozmod\/oniontx v.*/github.com\/kozmod\/oniontx $${val}/g" go.mod && \
			popd; \
		done; \
	fi)

.PHONY: help
help: ## List all make targets with description
	@grep -h -E '^[.a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: list
list: ## List all make targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort
