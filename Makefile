APP=orders

run:
	go run ./cmd/server

debug:
	go run -tags debug ./cmd/server

build:
	go build -o bin/$(APP) ./cmd/server