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
	@echo "  cache-init      инициализация volume orders-go-cache (одноразово, от root)"
	@echo "  build           go build -o bin/$(APP) ./cmd/server"
	@echo "  build-container go build в контейнере devcontainer (результат bin/$(APP))"
	@echo "  build-agent     go build -o tools/agent/build/orders-agent ./cmd/server"
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
# Контейнер запускается от текущего пользователя (--user), чтобы результаты
# сборки в workspace не принадлежали root. Кэши (GOMODCACHE/GOCACHE) живут
# в volume orders-go-cache, раздельно по /cache/mod и /cache/build.
check-container:
	docker run --rm -v "$$PWD":/workspace -w /workspace \
		-v orders-go-cache:/cache \
		-e GOMODCACHE=/cache/mod -e GOCACHE=/cache/build \
		--user "$$(id -u):$$(id -g)" \
		mcr.microsoft.com/devcontainers/go:1.26-bookworm \
		sh -c 'test -z "$$(gofmt -l .)" && go vet ./...'

test-container:
	docker run --rm -v "$$PWD":/workspace -w /workspace \
		-v orders-go-cache:/cache \
		-e GOMODCACHE=/cache/mod -e GOCACHE=/cache/build \
		--user "$$(id -u):$$(id -g)" \
		mcr.microsoft.com/devcontainers/go:1.26-bookworm \
		go test ./... -count=1

build:
	go build -o bin/$(APP) ./cmd/server

build-container:
	docker run --rm -v "$$PWD":/workspace -w /workspace \
		-v orders-go-cache:/cache \
		-e GOMODCACHE=/cache/mod -e GOCACHE=/cache/build \
		--user "$$(id -u):$$(id -g)" \
		mcr.microsoft.com/devcontainers/go:1.26-bookworm \
		sh -c 'mkdir -p bin && go build -buildvcs=false -o bin/$(APP) ./cmd/server'

cache-init:
	docker run --rm -v orders-go-cache:/cache \
		mcr.microsoft.com/devcontainers/go:1.26-bookworm \
		sh -c 'mkdir -p /cache/mod /cache/build && chown -R $(shell id -u):$(shell id -g) /cache'

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

.PHONY: help fmt vet test check verify cache-init check-container test-container build build-container build-agent run run-agent air clean clean-logs agent-clean
