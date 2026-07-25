package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Server works!")
	})

	log.Println("Listening on :3000")

	log.Fatal(http.ListenAndServe(":3000", nil))
}
