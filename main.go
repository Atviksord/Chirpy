package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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

// HELPER FUNCTIONS FOR JSONVALIDATEENDPOINT
func respondWithError(w http.ResponseWriter, code int, msg string) {

}
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {

}
func badWordReplacer(dirtybody string) string {

	badWordsMap := map[string]int{"kerfuffle": 1, "sharbert": 1, "fornax": 1}

	dirtybodyarray := strings.Split(dirtybody, " ")
	for i, word := range dirtybodyarray {
		_, exists := badWordsMap[strings.ToLower(word)]
		if exists {
			dirtybodyarray[i] = "****"
		}
	}
	return strings.Join(dirtybodyarray, " ")

}

func JsonValidateEndpoint(w http.ResponseWriter, r *http.Request, db *DB) {
	w.Header().Set("Content-Type", "application/json")

	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	// check for profanity and replace after this

	if err != nil {
		// an error will be thrown if the JSON is invalid or has the wrong types
		// any missing fields will simply have their values in the struct set to their zero value
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	type ValidResponse struct {
		CleanedBody string `json:"cleaned_body"`
	}

	type ErrorResponse struct {
		Error string `json:"error"`
	}

	errorResp := ErrorResponse{
		Error: "Chirp is too long",
	}
	cleaned_body := badWordReplacer(params.Body)
	validResp := ValidResponse{CleanedBody: cleaned_body}

	if len(params.Body) <= 140 { // IF OK
		w.WriteHeader(201)
		_, err := json.Marshal(validResp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}

		thefinal, err := db.CreateChirp(cleaned_body)
		if err != nil {
			fmt.Printf("Wow, couldnt get data from createchirp %v", err)
		}
		thefinalJson, err := json.Marshal(thefinal)
		if err != nil {
			fmt.Printf("Wow, couldnt marshal into finaljson %v", err)
		}
		w.Write(thefinalJson)

	} else if len(params.Body) > 140 { // IF ERROR
		w.WriteHeader(400)
		errordat, err := json.Marshal(errorResp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Write(errordat)

	}

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
func getChirpsHandler(w http.ResponseWriter, r *http.Request, db *DB) {
	cleanedChirps, err := db.GetChirps()
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		fmt.Printf("Error Loading Chirps from database %v", err)
		w.WriteHeader(500)
		return
	}
	cleanedChirpsJson, err := json.Marshal(cleanedChirps)
	if err != nil {
		fmt.Printf("Failed to turn chirps into json %v", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Write(cleanedChirpsJson)

}
func main() {
	apiCfg := &apiConfig{}

	mux := http.NewServeMux()
	// Initialize the database instance
	dbinstance, err := NewDB("database.json")
	if err != nil {
		log.Fatalf("Failed to initialize database: %s", err)
	}
	// ******** Handlerfunc registering station ************
	mux.HandleFunc("GET /api/healthz", CustomEndpoint) // registering a custom endpoint handler
	mux.HandleFunc("/api/reset", apiCfg.resetCounter)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsCounter)
	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		JsonValidateEndpoint(w, r, dbinstance)
	})
	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		getChirpsHandler(w, r, dbinstance)
	})
	mux.HandleFunc("GET /api/chirps")

	fileserver := http.FileServer(http.Dir("./static"))
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlecounterCors(fileserver)))

	corsMux := middlewareCors(mux)
	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: corsMux,
	}

	log.Fatal(server.ListenAndServe())

}
