package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type apiConfig struct { // struct to keep how many fileserverhits
	fileserverHits int
	JWT            string
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
	w.Header().Set("Content-Type", "application/json")
	cleanedChirps, err := db.GetChirps()
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
func getSpecificChirpsHandler(w http.ResponseWriter, r *http.Request, db *DB, id string) {
	ID, err := strconv.Atoi(id)
	if err != nil {
		fmt.Printf("Error converting ID to integer")
	}
	// loading file
	file, err := os.ReadFile(db.path)
	if err != nil {
		fmt.Printf("Error loading file %v", err)
	}
	var data DBStructure
	err = json.Unmarshal(file, &data)
	if err != nil {
		fmt.Printf("Error Unmarshalling file into struct")
	}
	// get chirp at ID
	specificchirp, exists := data.Chirps[ID]
	if !exists {
		w.WriteHeader(404)
		return
	}

	// marshall into json to write to site
	specialchirp, err := json.Marshal(specificchirp)
	if err != nil {
		fmt.Printf("Error marshalling into json to write to site %v", err)
	}

	w.WriteHeader(200)
	w.Write(specialchirp)

}

func addusers(w http.ResponseWriter, r *http.Request, db *DB) {
	// Add user, give ID and store email.
	w.Header().Set("Content-Type", "application/json")

	// get the reuqest body into params of User (json body)
	decoder := json.NewDecoder(r.Body)

	params := User{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Printf("Couldnt Decode request body %v", err)
	}
	// Create the User, assign dynamic ID and write to file
	returnvalue, err := db.CreateUser(params.Email, params.Password)

	if err != nil {
		fmt.Printf("Error marshalling byte data into json")
	}
	d, err := json.Marshal(returnvalue)
	if err != nil {
		fmt.Printf("Error creating user from email %v", err)
	}
	w.WriteHeader(201)
	w.Write(d)

}
func userlogin(w http.ResponseWriter, r *http.Request, db *DB) {
	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(r.Body)
	params := LoginRequest{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Printf("Couldnt Decode request body %v", err)
	}
	returnedUser, err := db.GetUser(params)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	// marshal returneduser back into json
	validuser, err := json.Marshal(returnedUser)
	if err != nil {
		fmt.Printf("Error marshalling returneduser into json data %v", err)
	}
	w.WriteHeader(200)
	w.Write(validuser)

}
func useredit(w http.ResponseWriter, r *http.Request, db *DB) {
	w.Header().Set("Content-Type", "application/json")
	// Decode the JSON from the request body and put it into a struct
	decoder := json.NewDecoder(r.Body)
	params := User{}
	err := decoder.Decode(&params)
	if err != nil {
		http.Error(w, "Couldnt Decode request body ", http.StatusBadRequest)
	}
	// Get the Authorization header
	authorization := r.Header.Get("Authorization")
	var tokenString string

	if !strings.HasPrefix(authorization, "Bearer ") {
		http.Error(w, "Unauthorized: missing or invalid token", http.StatusUnauthorized)
		return

	}
	// extract the tokenstring
	tokenString = strings.TrimPrefix(authorization, "Bearer ")

	// call the editUser function
	user, err := db.editUser(params, tokenString)
	if err != nil {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	// marshal the validUser to JSON and write the response
	validUser, err := json.Marshal(user)
	if err != nil {
		http.Error(w, "Failed to marshal user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(validUser)

}
func tokenRefreshHandler(w http.ResponseWriter, r *http.Request, db *DB) {
	// Get refresh token from header
	refreshTokenHeader := r.Header.Get("Authorization")

	validUserId, err := db.RefreshtokenCheck(refreshTokenHeader)
	if err != nil {
		w.WriteHeader(401)
	}

	// If found and valid, return newly generated JWT(not refresh)
	// Return the JWT 1 hour token in the responsewriter
	newJwtString, err := db.generateJWT(validUserId)
	if err != nil {
		fmt.Printf("Error generating JWT in tokenRefreshHandler: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return

	}
	tokenStruct := Tokenreturn{Token: newJwtString}

	jsonJwtString, err := json.Marshal(tokenStruct)
	if err != nil {
		fmt.Println("Cant marshal tokenstruct into json tknrefreshhandler")
	}
	w.Write(jsonJwtString)
	w.WriteHeader(200)

}
func tokenRevokeHandler(w http.ResponseWriter, r *http.Request, db *DB) {
	db.mux.Lock()
	defer db.mux.Unlock()

	refreshTokenHeader := r.Header.Get("Authorization")

	validUserId, err := db.RefreshtokenCheck(refreshTokenHeader)
	if err != nil {
		w.WriteHeader(401)
	}
	// Get DB to check
	datastructure, err := db.GetDatabase()
	if err != nil {
		fmt.Printf("Error getting DB, in TokenRevokeHandler %v", err)
	}
	validUser := datastructure.Users[validUserId]
	validUser.Refreshtoken = ""
	datastructure.Users[validUserId] = validUser

	file, err := json.Marshal(datastructure)
	err = os.WriteFile(db.path, file, 0644)
	if err != nil {
		fmt.Printf("Unable to write user to JSON file Tokenrevokehandler %v", err)
	}
	w.WriteHeader(204)

}

func main() {
	apiCfg := &apiConfig{}

	// JWT loading into apiConfig struct
	godotenv.Load()
	jwtSecret := os.Getenv("JWT_SECRET")
	apiCfg.JWT = jwtSecret

	dbg := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()
	if *dbg {
		fmt.Println("Debug mode is enabled!")
		err := os.Remove("database.json")
		if err != nil {
			fmt.Printf("Error deleting file %v", err)
		} else {
			fmt.Println("Database successfully deleted")
		}

	} else {
		fmt.Println("Debug mode is not enabled")
	}

	mux := http.NewServeMux()
	// Initialize the database instance
	dbinstance, err := NewDB("database.json", *apiCfg)
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
	mux.HandleFunc("GET /api/chirps/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		getSpecificChirpsHandler(w, r, dbinstance, id)
	})
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		addusers(w, r, dbinstance)
	})
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		userlogin(w, r, dbinstance)
	})
	mux.HandleFunc("PUT /api/users", func(w http.ResponseWriter, r *http.Request) {
		useredit(w, r, dbinstance)
	})
	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		tokenRefreshHandler(w, r, dbinstance)
	})
	mux.HandleFunc("PUT /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		tokenRevokeHandler(w, r, dbinstance)
	})

	fileserver := http.FileServer(http.Dir("./static"))
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlecounterCors(fileserver)))

	corsMux := middlewareCors(mux)
	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: corsMux,
	}

	log.Fatal(server.ListenAndServe())

}
