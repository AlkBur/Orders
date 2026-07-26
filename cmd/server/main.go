package main

import (
	"log"
	"os"

	"Orders/internal/app"
)

func main() {
	configPath := "config.json"
	if p := os.Getenv("ORDERS_CONFIG"); p != "" {
		configPath = p
	}

	a, err := app.New(configPath)
	if err != nil {
		log.Fatal(err)
	}

	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
