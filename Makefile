.DEFAULT_GOAL := help

WAILS ?= $(shell go env GOPATH)/bin/wails
BIN   := build/bin

.PHONY: help install build build-windows dev dev-web test lint clean

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

install: ## Install all dependencies (Go modules, Wails CLI, both frontends)
	go mod download
	go install github.com/wailsapp/wails/v2/cmd/wails@latest
	npm --prefix frontend install
	npm --prefix web install

build: ## Build the app for this machine (both UIs embedded) into build/bin/
	$(WAILS) build

build-windows: ## Build both UIs, then cross-compile a CGO-free Windows binary
	npm --prefix frontend run build
	npm --prefix web run build
	$(WAILS) build -platform windows/amd64 -s -skipbindings

dev: ## Run the admin window with hot reload
	$(WAILS) dev

dev-web: ## Run the guest UI dev server (proxies /api to the running guest server)
	npm --prefix web run dev

test: ## Run Go tests
	go test ./...

lint: ## gofmt check, go vet, and Vue type-check for both UIs
	@test -z "$$(gofmt -l . | tee /dev/stderr)" || (echo "gofmt: files need formatting" && exit 1)
	go vet ./...
	npm --prefix frontend run typecheck
	npm --prefix web run typecheck

clean: ## Remove build output and built UIs
	rm -rf $(BIN) frontend/dist/assets frontend/dist/index.html web/dist/assets web/dist/index.html
