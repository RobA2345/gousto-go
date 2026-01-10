package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	// The port you want to serve on
	port := ":8080"

	mux := http.NewServeMux()

	// Serve files from the current directory
	fs := http.FileServer(http.Dir("."))

	log.Printf("Gousto Archive serving on http://localhost%s\n", port)
	log.Println("Access from iPad using your Mac's IP (e.g., http://192.168.1.XX:8080)")

	// Wrap the file server with a "No-Cache" header
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
		w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
		w.Header().Set("Pragma", "no-cache")
		fs.ServeHTTP(w, r)
	})

	// Start the server
	err := http.ListenAndServe(port, mux)
	if err != nil {
		log.Fatal(err)
	}
}
