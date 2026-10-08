package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type contextKey string

const UserIdKey contextKey = "userID"

var ErrSessionEnded = errors.New("Session ended, please log in again")

var handlerExcludeList = []string{
	"/signup",
	"/login",
}

func (a *Api) authLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, s := range handlerExcludeList {
			if r.URL.Path == s {
				next.ServeHTTP(w, r)
				return
			}
		}

		cookie, err := r.Cookie("sid")

		if err != nil {
			http.Error(w, ErrSessionEnded.Error(), http.StatusUnauthorized)
			return
		}

		query := "SELECT expires_at, session_user_id FROM sessions WHERE session_id = $1 "

		row := a.db.QueryRow(query, cookie.Value)

		var expiresAt time.Time
		var userID int

		if scanErr := row.Scan(&expiresAt, &userID); scanErr != nil {
			if errors.Is(scanErr, sql.ErrNoRows) {
				http.Error(w, ErrSessionEnded.Error(), http.StatusUnauthorized)
			} else {
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
			}
			return
		}

		if expiresAt.Before(time.Now()) {
			http.Error(w, ErrSessionEnded.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIdKey, userID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	db := createDB()

	api := Api{
		addr: ":8081",
		db:   db,
	}

	mux := api.routes()

	server := http.Server{Addr: api.addr, Handler: mux}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
