package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Api struct {
	addr string
	db   *sql.DB
}

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	UserID   int    `json:"user_id"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

type ResponseUser struct {
	Email string `json:"email"`
	Name  string `json:"name"`
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

func (a *Api) loginUser(w http.ResponseWriter, r *http.Request) {
	var payload User

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// fetch id, email, password, in a row where email is payload.email.
	query := "SELECT user_id, email, password, name FROM users WHERE email= $1"

	row := a.db.QueryRow(query, payload.Email)

	u := User{}

	scanErr := row.Scan(&u.UserID, &u.Email, &u.Password, &u.Name)

	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			http.Error(w, "Invalid email, or password", http.StatusUnauthorized)
		} else {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
		}
		return
	}

	compareErr := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(payload.Password))

	if compareErr != nil {
		if errors.Is(compareErr, bcrypt.ErrMismatchedHashAndPassword) {
			http.Error(w, "Invalid email, or password", http.StatusUnauthorized)
		} else {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
		}
		return
	}

	respUser := ResponseUser{
		Name:  u.Name,
		Email: u.Email,
	}

	sessionQuery := "INSERT INTO sessions (session_user_id, session_id, expires_at) VALUES ($1, $2, $3)"

	sessionExpiry := time.Now().UTC().Add(time.Minute * 30)
	sessionID := uuid.NewString()

	_, dbErr := a.db.Exec(sessionQuery, u.UserID, sessionID, sessionExpiry)

	if dbErr != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "sid",
		Value:    sessionID,
		Expires:  sessionExpiry,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	safeEncode(respUser, http.StatusOK, w)
}

func (a *Api) createUser(w http.ResponseWriter, r *http.Request) {
	var payload User

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if ok := payload.Password == ""; ok {
		http.Error(w, "Password must not be empty", http.StatusBadRequest)
		return
	}

	if ok := payload.Name == ""; ok {
		http.Error(w, "Name must not be empty", http.StatusBadRequest)
		return
	}

	if ok := payload.Email == ""; ok {
		http.Error(w, "Email must not be empty", http.StatusBadRequest)
		return
	}

	u := ResponseUser{
		Email: payload.Email,
		Name:  payload.Name,
	}

	query := "INSERT INTO users (password, email, name) VALUES ($1, $2, $3)"

	password := payload.Password

	cost := 12

	hash, crypErr := bcrypt.GenerateFromPassword([]byte(password), cost)

	if crypErr != nil {
		http.Error(w, crypErr.Error(), http.StatusInternalServerError)
		return
	}

	email := u.Email
	name := u.Name

	_, dbErr := a.db.Exec(query, string(hash), email, name)

	if dbErr != nil {
		http.Error(w, dbErr.Error(), http.StatusInternalServerError)
		return
	}

	safeEncode(u, http.StatusCreated, w)
}

func (a *Api) createTask(w http.ResponseWriter, r *http.Request) {
	var payload Task

	defer r.Body.Close()

	if decodingErr := json.NewDecoder(r.Body).Decode(&payload); decodingErr != nil {
		http.Error(w, decodingErr.Error(), http.StatusBadRequest)
		return
	}

	if ok := payload.Title == ""; ok {
		http.Error(w, "Title cannot be empty", http.StatusBadRequest)
		return
	}

	newGUID := uuid.NewString()

	newTask := Task{
		ID:        newGUID,
		Title:     payload.Title,
		Done:      payload.Done,
		CreatedAt: time.Now(),
	}

	insertSQL := "INSERT INTO tasks (id, title, done, created_at) VALUES ($1, $2, $3, $4);"

	id := newTask.ID
	title := newTask.Title
	done := newTask.Done
	created_at := newTask.CreatedAt

	_, err := a.db.Exec(insertSQL, id, title, done, created_at)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	safeEncode(newTask, http.StatusCreated, w)
}

func (a *Api) getTasks(w http.ResponseWriter, r *http.Request) {
	query := "SELECT id, title, done, created_at FROM tasks"

	rows, err := a.db.Query(query)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	tasks := []Task{}

	for rows.Next() {
		var t Task

		err = rows.Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tasks = append(tasks, t)
	}

	if rows.Err() != nil {
		http.Error(w, rows.Err().Error(), http.StatusInternalServerError)
		return
	}

	safeEncode(tasks, http.StatusOK, w)
}

func (a *Api) getTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	query := "SELECT id, title, done, created_at FROM tasks WHERE id=$1"

	row := a.db.QueryRow(query, id)

	var t Task

	err := row.Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Task not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	safeEncode(t, http.StatusOK, w)
}
