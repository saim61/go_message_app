package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/saim61/go_message_app/internal/auth"
	"github.com/saim61/go_message_app/internal/storage/postgres"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

var (
	db    *sqlx.DB
	users *postgres.UserRepo
)

func main() {
	var err error
	// TODO: get username and password from env variables
	dsn := "postgres://postgres:postgres@localhost:5432/go_message_app?sslmode=disable"
	db, err = sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	users = postgres.NewUserRepo(db)

	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)

	addr := ":8080"
	log.Printf("auth service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := users.Create(r.Context(), &postgres.User{
		Username: req.Username,
		Password: hash,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	u, err := users.GetByUsername(r.Context(), req.Username)
	if err != nil || auth.CheckPassword(u.Password, req.Password) != nil {
		fmt.Println("*****************")
		fmt.Println("the error soniya:", err.Error())
		fmt.Println("*****************")
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	token, err := auth.NewToken(u.Username, 15*time.Minute)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(loginResponse{Token: token})
}
