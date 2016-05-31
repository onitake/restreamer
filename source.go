package main

import (
	"log"
	"net/http"
)

func main() {
	// Simple static webserver:
	log.Fatal(http.ListenAndServe("localhost:8000", http.FileServer(http.Dir("streams"))))
}
