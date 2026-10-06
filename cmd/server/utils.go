package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func createDB() *sql.DB {
	dotenvLoadErr := godotenv.Load()

	if dotenvLoadErr != nil {
		log.Fatal("Failed to load the dot env file")
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	// postgresql://<user>:<password>@<host>:<port>/<database>
	connString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", dbUser, dbPassword, dbHost, dbPort, dbName)

	db, dbErr := sql.Open("pgx", connString)

	if dbErr != nil {
		log.Fatal(dbErr)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	return db
}

func createTestDB(t *testing.T) *sql.DB {
	// helper func
	t.Helper()

	connString := "postgresql://postgres_test:password123_test@localhost:5434/app_test?sslmode=disable"

	db, err := sql.Open("pgx", connString)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}

	// defer
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func safeEncode(v any, succStatus int, w http.ResponseWriter) {
	var buffer bytes.Buffer

	if encodingErr := json.NewEncoder(&buffer).Encode(v); encodingErr != nil {
		http.Error(w, encodingErr.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(succStatus)
	w.Write(buffer.Bytes())
}
