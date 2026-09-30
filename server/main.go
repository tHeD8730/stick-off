package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func selfPing(url string) {
	for {
		time.Sleep(0.8 * time.Minute)
		http.Get(url)
	}
}

func main() {
	hub := newHub()
	go hub.run()

	mux := http.NewServeMux()

	// Serve the frontend
	mux.Handle("/", http.FileServer(http.Dir("./frontend")))

	// WebSocket endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(hub, w, r)
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","time":"` + time.Now().Format(time.RFC3339) + `"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	appUrl := os.Getenv("APP_URL)
	if appUrl != "" {
		go selfPing(appUrl + "/health")
	}

	log.Printf("Last Stick server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
