package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init() {
	var err error
	DB, err = sql.Open("sqlite3","./test.db")
	if err != nil {
		log.Fatal(err)
	}
}

func createUserTable() {
	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		uuid TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
	}
}	

func GetDetails(uuid string) (string, error) {
	var name string
	err := DB.QueryRow("SELECT name, email FROM users WHERE uuid = $1", uuid).Scan(&name)
	if err != nil {
		return "", "", err
	}
	return name, nil
}	
