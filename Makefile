.PHONY: run dev caddy build clean

run:
	go run ./cmd/server

dev:
	~/go/bin/air

caddy:
	caddy run --config Caddyfile

build:
	go build -o bin/receipt-server ./cmd/server

clean:
	rm -rf tmp bin