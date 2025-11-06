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
	@echo "Linting terraform files"
	@terraform fmt -check -diff -recursive

lint: lint-golangci-lint lint-tf

# DOCUMENTATION GENERATION
generate-docs:
	@echo "Generating documentation with tfplugindocs"
	@$(SCRIPTS_BASE)/tfplugindocs.sh

build:
	@go build -o bin/terraform-provider-stackit

fmt:
	@gofmt -s -w .
	@go tool goimports -w .
	@terraform fmt -diff -recursive

# TEST
test:
	@echo "Running tests for the terraform provider"
	@cd $(ROOT_DIR)/stackit && go test ./... -count=1 -coverprofile=coverage.out && cd $(ROOT_DIR)

# Test coverage
coverage:
	@echo ">> Creating test coverage report for the terraform provider"
	@cd $(ROOT_DIR)/stackit && (go test ./... -count=1 -coverprofile=coverage.out || true) && cd $(ROOT_DIR)
	@cd $(ROOT_DIR)/stackit && go tool cover -html=coverage.out -o coverage.html && cd $(ROOT_DIR)

test-acceptance-tf:
	@if [ -z $(TF_ACC_PROJECT_ID) ]; then echo "Input TF_ACC_PROJECT_ID missing"; exit 1; fi
	@if [ -z $(TF_ACC_ORGANIZATION_ID) ]; then echo "Input TF_ACC_ORGANIZATION_ID missing"; exit 1; fi
	@if [ -z $(TF_ACC_REGION) ]; then echo "Input TF_ACC_REGION missing"; exit 1; fi
	@if [ -z $(TF_ACC_TEST_IMAGE_LOCAL_FILE_PATH) ]; then \
		echo "Input TF_ACC_TEST_IMAGE_LOCAL_FILE_PATH missing. Creating a default file for testing."; \
	fi
	@echo "Running acceptance tests for the terraform provider"
	@cd $(ROOT_DIR)/stackit && TF_ACC=1 \
	TF_ACC_PROJECT_ID=$(TF_ACC_PROJECT_ID) \
	TF_ACC_ORGANIZATION_ID=$(TF_ACC_ORGANIZATION_ID) \
	TF_ACC_REGION=$(TF_ACC_REGION) \
	go test ./... -count=1 -timeout=30m && \
	cd $(ROOT_DIR)
