
 package main
 
 import (
     "log"
     "net/http"
     "os"
     "time"
 )
 
 func selfPing(url string) {
     for {
         time.Sleep(45 * time.Second)
         resp, err := http.Get(url)
         if err != nil {
             log.Printf("Self-ping failed: %v", err)
         } else {
             log.Printf("Self-ping %s %s", url, resp.Status)
             resp.Body.Close()
         }
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
 
     appUrl := os.Getenv("APP_URL")
     if appUrl != "" {
         go selfPing(appUrl + "/health")
     }
 
     log.Printf("Last Stick server listening on :%s", port)
     if err := http.ListenAndServe(":"+port, mux); err != nil {
         log.Fatal(err)
     }
 }
