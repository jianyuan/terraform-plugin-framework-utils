.DEFAULT_GOAL := help

##@ General
.PHONY: help
help: ## Show this help
	@awk 'BEGIN { FS = ":.*## " } \
	  /^##@ / { printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next } \
	  /^[a-zA-Z0-9_.-]+:.*## / { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ Development
.PHONY: generate
generate: ## Generate code and README documentation
	go generate ./...
	go tool gomarkdoc --repository.url https://github.com/jianyuan/terraform-plugin-framework-utils --repository.default-branch main --repository.path / --embed --output README.md ./...

.PHONY: test
test: ## Run unit tests with coverage
	go tool gotestsum --format github-actions -- -cover ./...
