.PHONT: tools
tools: ## Run tools (vet, gofmt, goimports, tidy, etc.)
	@go version
	gofmt -w .
	goimports -w .
	go mod tidy
	go vet ./...

.PHONY: list
list: ## List all make targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort
