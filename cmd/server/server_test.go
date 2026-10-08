package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testEmail = "test4@example.com"
var testPassword = "secret12345"

// Tasks tests

func TestCreateTask(t *testing.T) {
	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":0",
		db:   db,
	}

	task := Task{
		Title:       "TestTask",
		Description: "Test Description",
		Done:        false,
	}

	jsonData, jsonErr := json.Marshal(task)

	if jsonErr != nil {
		t.Fatal(jsonErr)
	}

	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	api.routes().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

// Users tests

func TestLogin(t *testing.T) {
	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":0",
		db:   db,
	}

	u := User{
		Email:    testEmail,
		Password: testPassword,
	}

	jsonData, jsonErr := json.Marshal(u)

	if jsonErr != nil {
		t.Fatal(jsonErr)
	}

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonData))

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	api.routes().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSignup(t *testing.T) {
	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":0",
		db:   db,
	}

	u := User{
		Name:     "testName",
		Email:    testEmail,
		Password: testPassword,
	}

	jsonData, jsonErr := json.Marshal(u)

	if jsonErr != nil {
		t.Fatal(jsonErr)
	}

	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	api.routes().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}
