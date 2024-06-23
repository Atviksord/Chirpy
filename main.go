package main

import (
	"fmt"
	"log"
	"net/http"
)

type apiConfig struct { // struct to keep how many fileserverhits
	fileserverHits int
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (cfg *apiConfig) middlecounterCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)

	})

}
func CustomEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
func (cfg *apiConfig) resetCounter(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.WriteHeader(http.StatusOK)
}
func (cfg *apiConfig) metricsCounter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	htmlResponse := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits)
	w.Write([]byte(htmlResponse))
}
func main() {
	apiCfg := &apiConfig{}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", CustomEndpoint) // registering a custom endpoint handler
	mux.HandleFunc("/api/reset", apiCfg.resetCounter)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsCounter)

	fileserver := http.FileServer(http.Dir("./static"))
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlecounterCors(fileserver)))

	corsMux := middlewareCors(mux)
	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: corsMux,
	}
	log.Fatal(server.ListenAndServe())

}
