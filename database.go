package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
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
	db := &DB{
		path: path,
		mux:  &sync.RWMutex{},
	}
	if err := db.ensureDB(); err != nil {
		return nil, err
	}
	return db, nil
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

	if data.Chirps == nil {
		data.Chirps = make(map[int]Chirp)
	}

	// Generate new ID based on length of Chirp Map in Dbstructure
	// Find the max existing ID and increment it
	maxID := 0
	for id := range data.Chirps {
		if id > maxID {
			maxID = id
		}
	}
	NewId := maxID + 1
	NewChirp := Chirp{Id: NewId, Body: body}

	// add chirp and id to map
	data.Chirps[NewId] = NewChirp

	// marshall data back to json
	validchirp, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error Error, Couldnt make Chirp")

	}
	err = os.WriteFile(db.path, validchirp, 0644)
	if err != nil {
		return Chirp{}, fmt.Errorf("error writing to database file: %w", err)
	}
	return NewChirp, err

}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	db.mux.Lock()
	defer db.mux.Unlock() // Lock to ensure safe concurrent access

	_, err := os.Stat(db.path)
	if os.IsNotExist(err) {
		initialData := DBStructure{
			Chirps: make(map[int]Chirp),
		}

		data, err := json.Marshal(initialData)
		if err != nil {
			return fmt.Errorf("error marshaling initial data: %v", err)
		}

		err = os.WriteFile(db.path, data, 0644)
		if err != nil {
			return fmt.Errorf("error writing initial data to file: %v", err)
		}

	} else if err != nil {
		return err
	}

	return nil
}

// use GET call to load chirps from database
func (db *DB) GetChirps() ([]Chirp, error) {
	data, err := os.ReadFile(db.path)
	if err != nil {
		fmt.Printf("Error loading file %v", err)
	}
	var Allchirps DBStructure
	err = json.Unmarshal(data, &Allchirps)
	if err != nil {
		fmt.Printf("Error Unmarshaling data %v", err)
	}
	var sortedChirps []Chirp

	for _, v := range Allchirps.Chirps {
		sortedChirps = append(sortedChirps, v)

	}
	sort.Slice(sortedChirps, func(j, i int) bool {
		return sortedChirps[i].Id > sortedChirps[j].Id
	})
	return sortedChirps, nil

}
