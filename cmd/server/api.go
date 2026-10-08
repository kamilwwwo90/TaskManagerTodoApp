package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
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

type UpdateObject struct {
	Change string `json:"change"`
	Value  any    `json:"value"`
}

func (a *Api) routes() http.Handler {
	mux := http.NewServeMux()

	// Users routing
	mux.HandleFunc("POST /signup", a.createUser)
	mux.HandleFunc("POST /login", a.loginUser)
	mux.HandleFunc("POST /logout", a.logoutUser)

	// Tasks routing
	mux.HandleFunc("GET /tasks", a.getTasks)
	mux.HandleFunc("POST /tasks", a.createTask)
	mux.HandleFunc("GET /tasks/{id}", a.getTask)
	mux.HandleFunc("PATCH /tasks/{id}", a.updateTask)
	mux.HandleFunc("DELETE /tasks/{id}", a.deleteTask)

	wrappedMux := a.authLogin(mux)

	return wrappedMux
}

func (a *Api) loginUser(w http.ResponseWriter, r *http.Request) {
	var payload User

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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

func (a *Api) logoutUser(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("sid")

	if err != nil {
		http.Error(w, ErrSessionEnded.Error(), http.StatusUnauthorized)
		return
	}

	sessionID := cookie.Value

	query := "DELETE FROM sessions WHERE session_id= $1"

	_, dbErr := a.db.Exec(query, sessionID)

	http.SetCookie(w, &http.Cookie{
		Name:     "sid",
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	if dbErr != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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

	query := "INSERT INTO users (password, email, name) VALUES ($1, $2, $3)"

	password := payload.Password

	cost := 12

	hash, crypErr := bcrypt.GenerateFromPassword([]byte(password), cost)

	if crypErr != nil {
		http.Error(w, crypErr.Error(), http.StatusInternalServerError)
		return
	}

	u := ResponseUser{
		Email: payload.Email,
		Name:  payload.Name,
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
		ID:          newGUID,
		Title:       payload.Title,
		Description: payload.Description,
		Done:        payload.Done,
		CreatedAt:   time.Now(),
	}

	query := "INSERT INTO tasks (user_id, id, description, title, done, created_at) VALUES ($1, $2, $3, $4, $5, $6)"

	user_id := r.Context().Value(UserIdKey)
	id := newTask.ID
	description := newTask.Description
	title := newTask.Title
	done := newTask.Done
	created_at := newTask.CreatedAt

	_, err := a.db.Exec(query, user_id, id, description, title, done, created_at)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	safeEncode(newTask, http.StatusCreated, w)
}

func (a *Api) getTasks(w http.ResponseWriter, r *http.Request) {
	query := "SELECT id, title, description, done, created_at FROM tasks WHERE user_id= $1"

	rows, err := a.db.Query(query, r.Context().Value(UserIdKey))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	tasks := []Task{}

	for rows.Next() {
		t := Task{}

		scanErr := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Done, &t.CreatedAt)

		if scanErr != nil {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
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

	query := "SELECT id, title, description, done, created_at FROM tasks WHERE id=$1 AND user_id= $2"

	row := a.db.QueryRow(query, id, r.Context().Value(UserIdKey))

	var t Task

	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Done, &t.CreatedAt)

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

func (a *Api) deleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")

	query := "DELETE FROM tasks WHERE id= $1 AND user_id= $2"

	result, dbErr := a.db.Exec(query, taskID, r.Context().Value(UserIdKey))

	if dbErr != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	rows, err := result.RowsAffected()

	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if rows == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

var whitelist = map[string]bool{
	"user_id":     false,
	"id":          false,
	"title":       true,
	"description": true,
	"done":        true,
	"created_at":  false,
}

func (a *Api) updateTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")

	var update UpdateObject

	jsonErr := json.NewDecoder(r.Body).Decode(&update)

	if jsonErr != nil {
		http.Error(w, jsonErr.Error(), http.StatusBadRequest)
		return
	}

	whitelisted, ok := whitelist[update.Change]

	if !ok {
		http.Error(w, "Invalid update field", http.StatusBadRequest)
		return
	}

	if !whitelisted {
		http.Error(w, "Attempted to modify a read only field", http.StatusBadRequest)
		return
	}

	query := fmt.Sprintf("UPDATE tasks SET %s = $1 WHERE id = $2 AND user_id = $3", update.Change)

	_, dbErr := a.db.Exec(query, update.Value, taskID, r.Context().Value(UserIdKey))

	if dbErr != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
