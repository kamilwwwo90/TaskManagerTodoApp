package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
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

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func createTask(t *testing.T, title string, description string, user User) string {
	t.Helper()

	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":8081",
		db:   db,
	}

	task := Task{
		Title:       title,
		Description: description,
	}

	jsonData, encodingErr := json.Marshal(task)

	if encodingErr != nil {
		t.Fatal(encodingErr)
	}

	cookie := renewSession(t, user)

	w := httptest.NewRecorder()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/tasks", bytes.NewBuffer(jsonData))

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cookie", cookie)

	api.routes().ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var payload Task

	decodingErr := json.NewDecoder(resp.Body).Decode(&payload)

	if decodingErr != nil {
		t.Fatal(decodingErr)
	}

	return payload.ID
}

func renewSession(t *testing.T, u User) string {
	t.Helper()

	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":8081",
		db:   db,
	}

	jsonData, encodingErr := json.Marshal(u)

	if encodingErr != nil {
		t.Fatal(encodingErr)
	}

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonData))

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	api.routes().ServeHTTP(w, req)

	cookie := w.Result().Header.Get("Set-Cookie")

	if cookie == "" {
		t.Fatal("expected auth cookie")
	}

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	return cookie
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
