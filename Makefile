ROOT_DIR              ?= $(shell git rev-parse --show-toplevel)
SCRIPTS_BASE          ?= $(ROOT_DIR)/scripts

# SETUP AND TOOL INITIALIZATION TASKS
project-help:
	@$(SCRIPTS_BASE)/project.sh help

project-tools:
	@$(SCRIPTS_BASE)/project.sh tools

# LINT
lint-golangci-lint:
	@echo "Linting with golangci-lint"
	@$(SCRIPTS_BASE)/lint-golangci-lint.sh

lint-tf: 
	@echo "Linting examples"
	@./bin/terraform fmt -check -diff -recursive examples

lint: lint-golangci-lint lint-tf

# DOCUMENTATION GENERATION
generate-docs:
	@echo "Generating documentation with tfplugindocs"
	@$(SCRIPTS_BASE)/tfplugindocs.sh

build:
	@go build -o bin/terraform-provider-stackit

fmt:
	@gofmt -s -w .

# TEST
test:
	@echo "Running tests for the terraform provider"
	@cd $(ROOT_DIR)/stackit && go test ./... -count=1 && cd $(ROOT_DIR)

test-acceptance-tf:
	@if [ -z $(TF_ACC_PROJECT_ID) ]; then echo "Input TF_ACC_PROJECT_ID missing"; exit 1; fi
	@echo "Running acceptance tests for the terraform provider"
	@cd $(ROOT_DIR)/stackit && TF_ACC=1 TF_ACC_PROJECT_ID=$(TF_ACC_PROJECT_ID) go test ./... -count=1 -timeout=30m && cd $(ROOT_DIR)

