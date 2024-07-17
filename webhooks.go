package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Polka struct {
	Event string `json:"event"`
	Data  struct {
		UserID int `json:"user_id"`
	} `json:"data"`
}

func (db *DB) polkachecker(p Polka) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	// check if its an upgrade

	if p.Event != "user.upgraded" {
		return nil
	}

	// GET DB
	userDB, err := db.GetDatabase()
	if err != nil {
		return fmt.Errorf("unable to get database")
	}
	// find matching ID on Polka
	userFound := false
	for _, user := range userDB.Users {
		if p.Data.UserID == user.Id {
			user.IsChirpyRed = true
			userDB.Users[user.Id] = user
			userFound = true
			break

		}
	}
	if !userFound {
		return fmt.Errorf("User not found")
	}
	// Write to file after update
	d, err := json.Marshal(userDB)
	if err != nil {
		fmt.Printf("Coiuldnt marshal data into json %v", err)

	}
	err = os.WriteFile(db.path, d, 0644)
	if err != nil {
		fmt.Printf("Unable to write user to JSON file %v", err)
	}

	return nil
}
