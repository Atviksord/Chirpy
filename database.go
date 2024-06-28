package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	// other necessary imports
)

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}
type DB struct {
	path string
	mux  *sync.RWMutex
}
type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	_, err := os.Stat(path) // quick existance check
	if os.IsNotExist(err) {
		initialData := `{"chirps": {}}`

		err := os.WriteFile(path, []byte(initialData), 0644)
		if err != nil {
			return nil, err
		}

	}
	return &DB{path: path, mux: &sync.RWMutex{}}, nil
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	db.mux.Lock()
	defer db.mux.Unlock() // lock and unlock to prevent race

	file, err := os.ReadFile(db.path) // fetch database file
	if err != nil {
		return Chirp{}, err
	}

	var data DBStructure // DatabaseJsonfile into a struct
	err = json.Unmarshal(file, &data)
	if err != nil {
		return Chirp{}, err
	}

	// Generate new ID based on length of Chirp Map in Dbstructure
	NewId := len(data.Chirps) + 1
	NewChirp := Chirp{Id: NewId, Body: body}

	// add chirp and id to map
	data.Chirps[NewId] = NewChirp

	// marshall data back to json
	validchirp, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error Error, Couldnt make Chirp")

	}
	os.WriteFile(db.path, validchirp, 0644)
	return NewChirp, err

}
