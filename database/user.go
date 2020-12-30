package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"golang.org/x/crypto/bcrypt"
)

// Credentials Create a struct that models the structure of a user, both in the request body, and in the DB
type Credentials struct {
	Password string `json:"password", db:"password"`
	Username string `json:"username", db:"username"`
	Email    string `json:"email", db:"email"`
}

// UserResponse - a user object that we can send as json
type UserResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

const registerUserQuery = `INSERT INTO membership.users values ($1, $2, $3)`
const getUserPasswordQuery = `SELECT password from membership.users where username=$1`
const getUserQuery = `SELECT username, email from membership.users where username=$1`

// RegisterUser register a user in the db
func (db *Database) RegisterUser(username string, password string, email string) error {
	if len(username) == 0 {
		return fmt.Errorf("not a valid user")
	}

	if len(password) == 0 {
		return fmt.Errorf("not a valid password")
	}

	if len(username) == 0 {
		return fmt.Errorf("not a valid email")
	}

	// Salt and hash the password using the bcrypt algorithm
	// The second argument is the cost of hashing, which we arbitrarily set as 8 (this value can be more or less, depending on the computing power you wish to utilize)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8)

	// Next, insert the username, along with the hashed password into the database
	rows, err := db.pool.Query(context.Background(), registerUserQuery, username, string(hashedPassword), email)
	if err != nil {
		return fmt.Errorf("conn.Query failed: %v", err)
	}

	defer rows.Close()

	return nil
}

// UserSignin - user login
func (db *Database) UserSignin(username string, password string) error {
	// We create another instance of `Credentials` to store the credentials we get from the database
	storedCreds := &Credentials{}

	// Get the existing entry present in the database for the given username
	row := db.pool.QueryRow(context.Background(), getUserPasswordQuery, username).Scan(&storedCreds.Password)
	if row == pgx.ErrNoRows {
		return fmt.Errorf("Unauthorized")
	}

	// Compare the stored hashed password, with the hashed version of the password that was received
	if err := bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(password)); err != nil {
		// If the two passwords don't match, return a 401 status
		return fmt.Errorf("Unauthorized: %s", err)
	}

	return nil
}

// GetUser returns the currently logged in user
func (db *Database) GetUser(username string) (UserResponse, error) {
	var userResponse UserResponse
	row := db.pool.QueryRow(context.Background(), getUserQuery, username).Scan(&userResponse.Username, &userResponse.Email)
	if row == pgx.ErrNoRows {
		return userResponse, fmt.Errorf("error getting user")
	}
	return userResponse, nil
}