package db

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(path string) {
	var err error
	DB, err = sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal(err)
	}

	if _, err = DB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Fatal(err)
	}

	createTables()
}

func createTables() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			uuid TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			log.Fatal(err)
		}
	}
}

type User struct {
	ID       int64
	UUID     string
	Name     string
	Email    string
	Password string
}

// CreateUser stores a new user. password must already be hashed.
func CreateUser(uuid, name, email, password string) (int64, error) {
	res, err := DB.Exec(
		`INSERT INTO users (uuid, name, email, password) VALUES (?, ?, ?, ?)`,
		uuid, name, email, password,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetUserByEmail returns sql.ErrNoRows if no user matches.
func GetUserByEmail(email string) (*User, error) {
	var u User
	err := DB.QueryRow(
		`SELECT id, uuid, name, email, password FROM users WHERE email = ?`,
		email,
	).Scan(&u.ID, &u.UUID, &u.Name, &u.Email, &u.Password)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func EmailExists(email string) (bool, error) {
	var n int
	err := DB.QueryRow(`SELECT COUNT(1) FROM users WHERE email = ?`, email).Scan(&n)
	return n > 0, err
}
