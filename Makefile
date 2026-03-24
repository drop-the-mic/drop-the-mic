PROJECT_DIR := $(shell pwd)
OPERATOR_DIR := $(PROJECT_DIR)/operator
UI_DIR := $(PROJECT_DIR)/ui
EMBED_DIR := $(OPERATOR_DIR)/server/embed/dist

# Image settings
IMG_OPERATOR ?= ghcr.io/drop-the-mic/operator:latest
IMG_SERVER ?= ghcr.io/drop-the-mic/server:latest

.PHONY: all
all: generate manifests lint test build

##@ Development

.PHONY: generate
generate: ## Generate deepcopy and other code
	cd $(OPERATOR_DIR) && make generate

.PHONY: manifests
manifests: ## Generate CRD manifests
	cd $(OPERATOR_DIR) && make manifests

.PHONY: lint
lint: ## Run linters
	cd $(OPERATOR_DIR) && golangci-lint run ./...

.PHONY: test
test: ## Run unit + integration tests
	cd $(OPERATOR_DIR) && go test ./... -v

##@ Build

.PHONY: ui-build
ui-build: ## Build React UI
	cd $(UI_DIR) && npm ci && npm run build
	rm -rf $(EMBED_DIR)
	cp -r $(UI_DIR)/dist $(EMBED_DIR)

.PHONY: build
build: ui-build ## Build operator and server binaries
	cd $(OPERATOR_DIR) && go build -o bin/operator ./cmd/main.go
	cd $(OPERATOR_DIR) && go build -o bin/server ./server/main.go

.PHONY: build-operator
build-operator: ## Build operator binary only
	cd $(OPERATOR_DIR) && go build -o bin/operator ./cmd/main.go

.PHONY: build-server
build-server: ui-build ## Build server binary with embedded UI
	cd $(OPERATOR_DIR) && go build -o bin/server ./server/main.go

##@ Docker

.PHONY: docker-build
docker-build: ## Build Docker images
	docker build -f Dockerfile.operator -t $(IMG_OPERATOR) .
	docker build -f Dockerfile.server -t $(IMG_SERVER) .

.PHONY: docker-push
docker-push: ## Push Docker images
	docker push $(IMG_OPERATOR)
	docker push $(IMG_SERVER)

##@ Helm

.PHONY: helm-package
helm-package: manifests ## Package Helm chart
	helm package charts/drop-the-mic

.PHONY: helm-install
helm-install: ## Install Helm chart
	helm install dtm charts/drop-the-mic

.PHONY: helm-upgrade
helm-upgrade: ## Upgrade Helm chart
	helm upgrade dtm charts/drop-the-mic

##@ Local Development

.PHONY: dev
dev: manifests ## Deploy to local kind cluster
	kind create cluster --name dtm 2>/dev/null || true
	kubectl apply -f $(OPERATOR_DIR)/config/crd/bases/
	cd $(OPERATOR_DIR) && go run ./cmd/main.go

.PHONY: dev-ui
dev-ui: ## Run UI dev server
	cd $(UI_DIR) && npm run dev

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf $(OPERATOR_DIR)/bin
	rm -rf $(UI_DIR)/dist
	rm -rf $(EMBED_DIR)

##@ Help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
