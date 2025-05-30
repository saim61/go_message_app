package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/saim61/go_message_app/internal/auth"
	"github.com/saim61/go_message_app/internal/storage/postgres"
	"github.com/saim61/go_message_app/utils"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

var users *postgres.UserRepo

func main() {
	_ = godotenv.Load() // loads .env if present (noop in Docker)

	dsn := utils.BuildDSN()
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	users = postgres.NewUserRepo(db)

	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)

	port := utils.GetEnv("AUTH_PORT", "8080")
	log.Printf("[auth] listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	hash, _ := auth.HashPassword(req.Password)
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
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	u, err := users.GetByUsername(r.Context(), req.Username)
	if err != nil || auth.CheckPassword(u.Password, req.Password) != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	token, _ := auth.NewToken(u.Username, 15*time.Minute)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(loginResponse{Token: token})
}
