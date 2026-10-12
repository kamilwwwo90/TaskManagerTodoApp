package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testName = "testName"
var testEmail = "test4@example.com"
var testPassword = "secret12345"

var testTitle = "testTitle"
var testDescription = "testDescription"

// Tasks tests

func TestGetTasks(t *testing.T) {
	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":8081",
		db:   db,
	}

	u := User{
		Email:    testEmail,
		Password: testPassword,
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/tasks", nil)

	cookie := renewSession(t, u)

	req.Header.Add("Cookie", cookie)

	w := httptest.NewRecorder()

	api.routes().ServeHTTP(w, req)

	var payload []Task

	decodingErr := json.NewDecoder(w.Result().Body).Decode(&payload)

	if decodingErr != nil {
		t.Fatal(decodingErr)
	}

	t.Log(payload)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTask(t *testing.T) {
	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":8081",
		db:   db,
	}

	u := User{
		Email:    testEmail,
		Password: testPassword,
	}

	update := UpdateObject{
		Change: "title",
		Value:  "anotherTitle",
	}

	taskID := createTask(t, testTitle, testDescription, u)

	jsonData, encodingErr := json.Marshal(update)

	if encodingErr != nil {
		t.Fatal(encodingErr)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/tasks/"+taskID, bytes.NewBuffer(jsonData))

	cookie := renewSession(t, u)

	req.Header.Add("Cookie", cookie)

	w := httptest.NewRecorder()

	api.routes().ServeHTTP(w, req)

	query := "SELECT title, description FROM tasks WHERE id= $1"

	row := api.db.QueryRow(query, taskID)

	var task Task

	scanErr := row.Scan(&task.Title, &task.Description)

	if scanErr != nil {
		t.Fatal(scanErr)
	}

	t.Log(task)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetTask(t *testing.T) {
	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":8081",
		db:   db,
	}

	u := User{
		Email:    testEmail,
		Password: testPassword,
	}

	taskID := createTask(t, testTitle, testDescription, u)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/tasks/"+taskID, nil)

	cookie := renewSession(t, u)

	req.Header.Add("Cookie", cookie)

	w := httptest.NewRecorder()

	api.routes().ServeHTTP(w, req)

	var payload Task

	decodingErr := json.NewDecoder(w.Result().Body).Decode(&payload)

	if decodingErr != nil {
		t.Fatal(decodingErr)
	}

	t.Log(payload)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTask(t *testing.T) {
	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":8081",
		db:   db,
	}

	u := User{
		Email:    testEmail,
		Password: testPassword,
	}

	taskID := createTask(t, testTitle, testDescription, u)

	w := httptest.NewRecorder()

	cookie := renewSession(t, u)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/tasks/"+taskID, nil)

	req.Header.Add("Cookie", cookie)

	api.routes().ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateTask(t *testing.T) {
	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":8081",
		db:   db,
	}

	task := Task{
		Title:       "Test1234",
		Description: "Test1234",
		Done:        false,
	}

	jsonData, jsonErr := json.Marshal(task)

	if jsonErr != nil {
		t.Fatal(jsonErr)
	}

	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBuffer(jsonData))

	cookie := renewSession(t, User{
		Email:    testEmail,
		Password: testPassword,
	})

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()

	api.routes().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

// Users tests

func TestLogout(t *testing.T) {
	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":8081",
		db:   db,
	}

	u := User{
		Email:    testEmail,
		Password: testPassword,
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/logout", nil)

	cookie := renewSession(t, u)

	req.Header.Add("Cookie", cookie)

	w := httptest.NewRecorder()

	api.routes().ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin(t *testing.T) {
	db := createTestDB(t)

	defer db.Close()

	api := Api{
		addr: ":8081",
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
		addr: ":8081",
		db:   db,
	}

	u := User{
		Name:     testName,
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
