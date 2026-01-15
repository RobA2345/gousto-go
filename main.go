package main

import (
	"log"
	"net/http"
)

func main() {
	port := ":8080"

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("."))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})

	err := http.ListenAndServe(port, mux)
	if err != nil {
		log.Fatal(err)
	}
}
