package main

import (
	"log"
	"net/http"
)

func main() {
	// The port you want to serve on
	port := ":8080"

	mux := http.NewServeMux()

	// Serve files from the current directory
	fs := http.FileServer(http.Dir("."))

	// Wrap the file server with a "No-Cache" header
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})

	// Start the server
	err := http.ListenAndServe(port, mux)
	if err != nil {
		log.Fatal(err)
	}
}
