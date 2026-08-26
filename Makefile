APP=orders

.DEFAULT_GOAL := help

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Development:"
	@echo "  fmt         gofmt -l -w"
	@echo "  vet         go vet ./..."
	@echo "  test        go test ./... -count=1"
	@echo "  check       fmt + vet"
	@echo "  verify      check + test"
	@echo "  check-container  gofmt + vet в контейнере devcontainer"
	@echo "  test-container   go test в контейнере devcontainer"
	@echo ""
	@echo "Build:"
	@echo "  build       go build -o bin/$(APP) ./cmd/server"
	@echo "  build-agent go build -o tools/agent/build/orders-agent ./cmd/server"
	@echo ""
	@echo "Run:"
	@echo "  run         go run ./cmd/server"
	@echo "  run-agent   ORDERS_CONFIG=config.agent.json go run ./cmd/server"
	@echo "  air         ~/go/bin/air"
	@echo ""
	@echo "Clean:"
	@echo "  clean       rm -rf bin/ tools/agent/build/ tools/agent/temp/*"
	@echo "  clean-logs  rm -rf tools/agent/logs/*"
	@echo "  agent-clean rm -rf tools/agent/temp/* tools/agent/reports/*"

fmt:
	gofmt -l -w .

vet:
	go vet ./...

test:
	go test ./... -count=1

check: fmt vet

verify: check test

# Контейнерная проверка (без установки Go в хост-среду).
# Образ из .devcontainer/devcontainer.json; кэш в именованном volume.
check-container:
	docker run --rm -v "$$PWD":/workspace -w /workspace \
		-v orders-go-cache:/go \
		mcr.microsoft.com/devcontainers/go:1.26-bookworm \
		sh -c 'test -z "$$(gofmt -l .)" && go vet ./...'

test-container:
	docker run --rm -v "$$PWD":/workspace -w /workspace \
		-v orders-go-cache:/go \
		mcr.microsoft.com/devcontainers/go:1.26-bookworm \
		go test ./... -count=1

build:
	go build -o bin/$(APP) ./cmd/server

build-agent:
	go build -o tools/agent/build/orders-agent ./cmd/server

run:
	go run ./cmd/server

run-agent:
	ORDERS_CONFIG=config.agent.json go run ./cmd/server

air:
	~/go/bin/air

clean:
	rm -rf bin/ tools/agent/build/ tools/agent/temp/*

clean-logs:
	rm -rf tools/agent/logs/*

agent-clean:
	rm -rf tools/agent/temp/* tools/agent/reports/*

.PHONY: help fmt vet test check verify check-container test-container build build-agent run run-agent air clean clean-logs agent-clean
