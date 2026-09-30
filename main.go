package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

var ErrSessionEnded = errors.New("Session ended, please log in again")

var handlerExcludeList = []string{
	"/Signup",
	"/Login",
}

func (a *Api) authLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, s := range handlerExcludeList {
			if r.URL.Path == s {
				next.ServeHTTP(w, r)
				return
			}
		}

		//Code executed before

		cookie, err := r.Cookie("sid")

		// error means the browser did not send it, therefore it must've expired.
		if err != nil {
			http.Error(w, ErrSessionEnded.Error(), http.StatusUnauthorized)
			return
		}

		// give me session id from sessions, where it matches cookie.value (bless you pgsql)
		query := "SELECT expires_at FROM sessions WHERE session_id = $1 "

		row := a.db.QueryRow(query, cookie.Value)

		var expiresAt time.Time // TIMESTAMPTZ

		// check if row exists, and get the session id
		if scanErr := row.Scan(&expiresAt); scanErr != nil {
			if errors.Is(scanErr, sql.ErrNoRows) {
				http.Error(w, ErrSessionEnded.Error(), http.StatusUnauthorized)
			} else {
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
			}
			return
		}

		// So if it got to this point in the code, then the cookie exists, and there is a session ID that matches it's value.

		if expiresAt.Before(time.Now()) {
			http.Error(w, ErrSessionEnded.Error(), http.StatusUnauthorized)
			return
		}

		// Session is valid, safe to move onto the main handler logic

		next.ServeHTTP(w, r)

		// Code executed after

		// Here I can probably check for statistics about the routing, and such
	})
}

func main() {
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

	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	api := Api{
		addr: ":8081",
		db:   db,
	}

	mux := http.NewServeMux()

	// Tasks routing
	mux.HandleFunc("POST /Tasks", api.createTask)
	mux.HandleFunc("GET /Tasks", api.getTasks)

	mux.HandleFunc("GET /Tasks/{id}", api.getTask)

	// Users routing
	mux.HandleFunc("POST /Signup", api.createUser)
	mux.HandleFunc("POST /Login", api.loginUser)

	wrappedMux := api.authLogin(mux)

	server := http.Server{Addr: api.addr, Handler: wrappedMux}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
