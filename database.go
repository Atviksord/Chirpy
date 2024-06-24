package main

import (
	"sync"
	// other necessary imports
)

type Chirp struct {
	id   int
	body string
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
	return nil
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	return nil
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	return nil
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	return nil
}

// loadDB reads the database file into memory
func (db *DB) loadDB() (DBStructure, error) {

	return nil

}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStructure DBStructure) error {

	return nil
}
