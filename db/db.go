package db

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init(dsn string) {
	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Compose starts the app as soon as Postgres accepts connections, which
	// is a moment before it is ready to serve. Retry rather than crash.
	for i := 0; i < 10; i++ {
		if err = DB.Ping(); err == nil {
			break
		}
		log.Printf("waiting for postgres (%d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("could not reach postgres: %v", err)
	}

	createTables()
}

func createTables() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			uuid TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS items (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_items_created ON items(created_at DESC)`,
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
	// Postgres has no LastInsertId; the id comes back from RETURNING.
	var id int64
	err := DB.QueryRow(
		`INSERT INTO users (uuid, name, email, password)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		uuid, name, email, password,
	).Scan(&id)
	return id, err
}

// GetUserByEmail returns sql.ErrNoRows if no user matches.
func GetUserByEmail(email string) (*User, error) {
	var u User
	err := DB.QueryRow(
		`SELECT id, uuid, name, email, password FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.UUID, &u.Name, &u.Email, &u.Password)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func EmailExists(email string) (bool, error) {
	var n int
	err := DB.QueryRow(`SELECT COUNT(1) FROM users WHERE email = $1`, email).Scan(&n)
	return n > 0, err
}
