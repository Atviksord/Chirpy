package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"
)

type ExpireUser struct {
	Expires_in_seconds int `json:"expires_in_seconds"`
}
type Tokenreturn struct {
	Token string `json:"token"`
}
type User struct {
	Id            int       `json:"id"`
	Email         string    `json:"email"`
	Password      string    `json:"password"`
	Refreshtoken  string    `json:"refresh_token"`
	Refreshexpiry time.Time `json:"refresh_token_expiry"`
	IsChirpyRed   bool      `json:"is_chirpy_red"`
}
type responseUser struct {
	Id           int    `json:"id"`
	Email        string `json:"email"`
	Token        string `json:"token"`
	Refreshtoken string `json:"refresh_token"`
	IsChirpyRed  bool   `json:"is_chirpy_red"`
}

type Chirp struct {
	Id        int    `json:"id"`
	Body      string `json:"body"`
	Author_id int    `json:"author_id"`
}
type DB struct {
	path   string
	mux    *sync.RWMutex
	config apiConfig
}
type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users  map[int]User  `json:"users"`
}
type LoginRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds *int   `json:"expires_in_seconds"` // Optional
}

func (db *DB) RefreshtokenCheck(refreshTokenHeader string) (int, error) {

	// Check if Header string starts with Bearer
	if !strings.HasPrefix(refreshTokenHeader, "Bearer ") {
		return 0, fmt.Errorf("unauthorized: missing or invalid token")

	}
	// extract the tokenstring

	tokenString := strings.TrimPrefix(refreshTokenHeader, "Bearer ")
	// get DB
	userData, err := db.GetDatabase()
	if err != nil {
		return 0, fmt.Errorf("error returning DB, in refreshhandler %v", err)

	}

	// Loop DB for Refresh token match
	var found bool
	var validUserId int
	for _, user := range userData.Users {
		if user.Refreshtoken == tokenString {
			found = true
			if user.Refreshexpiry.Before(time.Now().UTC()) {
				// If refresh token has expired
				return 0, fmt.Errorf("refresh token has expired")

			}
			validUserId = user.Id
		}

	}
	if !found {
		return 0, fmt.Errorf("refresh token not found no match")

	}
	return validUserId, nil

}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string, apiCfg apiConfig) (*DB, error) {
	db := &DB{
		path:   path,
		mux:    &sync.RWMutex{},
		config: apiCfg,
	}
	if err := db.ensureDB(); err != nil {
		return nil, err
	}
	return db, nil
}

// CreateUser creates a user from a post request
func (db *DB) CreateUser(username string, password string) (responseUser, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	// fetch database json file
	file, err := os.ReadFile(db.path) // fetch database file
	if err != nil {
		return responseUser{}, err
	}

	// make an instance of DB
	usersmap := DBStructure{}
	err = json.Unmarshal(file, &usersmap)
	if err != nil {
		return responseUser{}, err
	}
	// create usermap if it doesnt exist
	if usersmap.Users == nil {
		usersmap.Users = make(map[int]User)
	}
	// check for max ID by checking the map
	maxID := 0
	for _, user := range usersmap.Users {
		if user.Id > maxID {
			maxID = user.Id
		}
	}
	NewId := maxID + 1

	// ENCRYPT PASSWORD
	hashedpass, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		fmt.Println("ERROR HASHING PASSWORD")
		return responseUser{}, err
	}

	currentUser := User{Email: username, Id: NewId, Password: string(hashedpass)}
	// change the User struct into Json with dynamic ID added

	// add User object struct to the map
	usersmap.Users[NewId] = currentUser
	d, err := json.Marshal(usersmap)

	if err != nil {
		fmt.Printf("Couldnt marshal data into json %v", err)

	}
	err = os.WriteFile(db.path, d, 0644)
	if err != nil {
		fmt.Printf("Unable to write user to JSON file %v", err)
	}

	// return struct reply without password
	returnUser := responseUser{Email: currentUser.Email, Id: currentUser.Id}
	return returnUser, err

}

func (db *DB) GetDatabase() (DBStructure, error) {
	file, err := os.ReadFile(db.path) // fetch database file
	if err != nil {
		return DBStructure{}, err
	}

	// make instance of the DBSTRUCTURE to dump raw data
	datastructure := DBStructure{}

	err = json.Unmarshal(file, &datastructure)
	if err != nil {
		fmt.Printf("Failed to dump raw json data into struct %v", err)
	}
	return datastructure, nil

}
func (db *DB) generateJWT(userID int) (string, error) {
	// My JWT creation logic here, using the userID and email
	// 1 Hour expiration
	currentTime := jwt.NewNumericDate(time.Now().UTC())
	duration := time.Duration(3600) * time.Second
	expirationTime := time.Now().UTC().Add(duration)
	jwtExpirationTime := jwt.NewNumericDate(expirationTime)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  currentTime,
		ExpiresAt: jwtExpirationTime,
		Subject:   strconv.Itoa(userID),
	})

	signedToken, err := token.SignedString([]byte(db.config.JWT))
	if err != nil {
		fmt.Printf("Failed to sign token %v", err)
		return "", err
	}
	return signedToken, nil

}

// login function,  look up user, auth and log in.
func (db *DB) GetUser(u LoginRequest) (responseUser, error) {
	db.mux.Lock()
	defer db.mux.Unlock()
	// load DB file into datastructure struct
	datastructure, err := db.GetDatabase()
	if err != nil {
		fmt.Println("Failed to load Database")
		return responseUser{}, err
	}

	// Find user by email, if not found return error
	var targetuser User
	var found bool
	var index int
	for i, user := range datastructure.Users {
		if u.Email == user.Email {
			index = i
			targetuser = user
			found = true
			break

		}

	}
	if !found {
		fmt.Println("User matching the email not found")
		return responseUser{}, err
	}

	// Compare password of Dbuser with Parameter user hashed password for match
	err = bcrypt.CompareHashAndPassword([]byte(targetuser.Password), []byte(u.Password))
	if err != nil {
		fmt.Printf("Wrong password %v", err)
		return responseUser{}, err
	}
	// Check for Optional parameter time out seconds
	if u.ExpiresInSeconds == nil {
		u.ExpiresInSeconds = new(int)
		*u.ExpiresInSeconds = 3600
	} else if *u.ExpiresInSeconds > 3600 {
		*u.ExpiresInSeconds = 3600
	}
	currentTime := jwt.NewNumericDate(time.Now().UTC())
	duration := time.Duration(*u.ExpiresInSeconds) * time.Second
	expirationTime := time.Now().UTC().Add(duration)
	jwtExpirationTime := jwt.NewNumericDate(expirationTime)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  currentTime,
		ExpiresAt: jwtExpirationTime,
		Subject:   strconv.Itoa(targetuser.Id),
	})

	signedToken, err := token.SignedString([]byte(db.config.JWT))
	if err != nil {
		fmt.Printf("Failed to sign token %v", err)
	}

	// Generate Refresh Token
	// Generate Refresh Token
	refreshTokenBytes := make([]byte, 32) // 256 bits
	_, err = rand.Read(refreshTokenBytes)
	if err != nil {
		fmt.Printf("Failed to generate refresh token: %v", err)
	}
	refreshCode := hex.EncodeToString(refreshTokenBytes)
	// Get refreshtoken expiry time
	refreshExpiry := time.Now().UTC().Add(time.Hour * 24 * 60)

	// writing refresh token to DB
	targetuser.Refreshtoken = refreshCode
	targetuser.Refreshexpiry = refreshExpiry

	datastructure.Users[index] = targetuser
	updatedData, err := json.Marshal(datastructure)
	if err != nil {
		return responseUser{}, fmt.Errorf("could not marshal data into JSON: %v", err)
	}
	err = os.WriteFile(db.path, updatedData, 0644)
	if err != nil {
		fmt.Printf("Unable to write user to JSON file %v", err)
	}

	responseTarget := responseUser{Email: targetuser.Email, Id: targetuser.Id, Token: signedToken, Refreshtoken: refreshCode, IsChirpyRed: targetuser.IsChirpyRed}

	return responseTarget, nil
}
func (db *DB) editUser(u User, token string) (responseUser, error) {
	// lock and unlock
	db.mux.Lock()
	defer db.mux.Unlock()

	// Validate the token to allow update
	validToken, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(db.config.JWT), nil
	})
	if err != nil {
		return responseUser{}, fmt.Errorf("invalid token %v", err)
	}
	if !validToken.Valid {
		return responseUser{}, fmt.Errorf("could not exctract claims: %v", err)
	}

	// extract claims
	claims, ok := validToken.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return responseUser{}, fmt.Errorf(
			"error converting stringified ID to int %v",
			err,
		)
	}

	// get valid ID from claims
	stringID := claims.Subject
	validID, err := strconv.Atoi(stringID)
	if err != nil {
		fmt.Printf("Error getting subject from claims: %v\n", err)
		return responseUser{}, err
	}

	// Fetch User from database
	datastructure, err := db.GetDatabase()
	if err != nil {
		return responseUser{}, fmt.Errorf("failed to load database: %v", err)
	}

	// Hashing password before updating database
	// ENCRYPT PASSWORD
	hashedpass, err := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
	if err != nil {
		fmt.Println("ERROR HASHING PASSWORD")
		return responseUser{}, err
	}
	u.Password = string(hashedpass)

	// rewriting database with updated user
	userToUpdate := datastructure.Users[validID]
	if u.Email != "" {
		userToUpdate.Email = u.Email

	}
	if u.Password != "" {
		userToUpdate.Password = u.Password

	}
	datastructure.Users[validID] = userToUpdate

	updatedData, err := json.Marshal(datastructure)

	if err != nil {
		return responseUser{}, fmt.Errorf("could not marshal data into JSON: %v", err)

	}
	// Write updated data back to the json database
	err = os.WriteFile(db.path, updatedData, 0644)
	if err != nil {
		fmt.Printf("Unable to write user to JSON file %v", err)
	}
	return responseUser{Email: u.Email, Id: validID}, nil

}

func (db *DB) JwtValidationCheck(token string) (int, error) {
	// Validate the token to allow update
	validToken, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(db.config.JWT), nil
	})
	if err != nil {
		return 0, fmt.Errorf("invalid token %v", err)
	}
	if !validToken.Valid {
		return 0, fmt.Errorf("could not exctract claims: %v", err)
	}

	// extract claims
	claims, ok := validToken.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return 0, fmt.Errorf(
			"error converting stringified ID to int %v",
			err,
		)
	}

	// get valid ID from claims
	stringID := claims.Subject
	validID, err := strconv.Atoi(stringID)
	if err != nil {
		fmt.Printf("Error getting subject from claims: %v\n", err)
		return 0, err
	}
	return validID, nil

}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string, authorid int) (Chirp, error) {
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
	NewChirp := Chirp{Id: NewId, Body: body, Author_id: authorid}

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
func (db *DB) GetAuthorChirps(chirps []Chirp, authorId string) ([]Chirp, error) {
	var trueChirp []Chirp

	authorIdz, err := strconv.Atoi(authorId)
	if err != nil {
		fmt.Print("Couldnt return")
	}

	for _, chirp := range chirps {
		if chirp.Author_id == authorIdz {
			trueChirp = append(trueChirp, chirp)
		}
	}
	sort.Slice(trueChirp, func(j, i int) bool {
		return trueChirp[i].Id > trueChirp[j].Id
	})
	return trueChirp, nil
}
func (db *DB) MasterSorter(chirps []Chirp, authorId string, sorter string) ([]Chirp, error) {
	if (sorter == "asc") || (sorter == "") {
		sort.Slice(chirps, func(j, i int) bool {
			return chirps[i].Id > chirps[j].Id
		})

	}
	if sorter == "desc" {
		sort.Slice(chirps, func(j, i int) bool {
			return chirps[i].Id < chirps[j].Id
		})

	}
	return chirps, nil

}
