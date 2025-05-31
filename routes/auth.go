package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/saim61/go_message_app/internal/auth"
	"github.com/saim61/go_message_app/internal/httpx"
	"github.com/saim61/go_message_app/internal/storage/postgres"
)

type AuthRoutes struct {
	users *postgres.UserRepo
}

func RegisterAuth(r *gin.RouterGroup, db *sqlx.DB) {
	ar := &AuthRoutes{users: postgres.NewUserRepo(db)}
	r.POST("/register", ar.register)
	r.POST("/login", ar.login)
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type loginResponse struct {
	Token string `json:"token"`
}

func (a *AuthRoutes) register(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpx.Fail("Bad values for registering User", http.StatusBadRequest))
		return
	}
	hash, _ := auth.HashPassword(req.Password)
	if err := a.users.Create(c, &postgres.User{
		Username: req.Username,
		Password: hash,
	}); err != nil {
		c.JSON(http.StatusConflict, httpx.Fail(err.Error(), http.StatusConflict))
		return
	}
	c.JSON(http.StatusCreated, httpx.OK("Registration Successful! Please proceed to login to get JWT.", struct{}{}))
}

func (a *AuthRoutes) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpx.Fail("Bad values for login User", http.StatusBadRequest))
		return
	}
	u, err := a.users.GetByUsername(c, req.Username)
	if err != nil || auth.CheckPassword(u.Password, req.Password) != nil {
		c.JSON(http.StatusUnauthorized, httpx.Fail("Invalid credentials", http.StatusUnauthorized))
		return
	}
	tok, _ := auth.NewToken(u.Username, 15*time.Minute)
	c.JSON(http.StatusOK, httpx.OK("Successfully login", loginResponse{Token: tok}))
}
