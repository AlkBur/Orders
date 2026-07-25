package main

import (
	"log"

	"orders/internal/app"
)

func main() {

	a, err := app.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
