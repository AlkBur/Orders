APP=orders

.DEFAULT_GOAL := air

run:
	go run ./cmd/server

air:
	~/go/bin/air

debug:
	~/go/bin/air

build:
	go build -o bin/$(APP) ./cmd/server