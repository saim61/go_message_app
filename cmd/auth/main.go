package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/saim61/go_message_app/routes"
	"github.com/saim61/go_message_app/utils"
)

func main() {
	_ = godotenv.Load()
	dsn := utils.BuildDSN()
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()
	router.Use(cors.Default())
	api := router.Group("/api/v1")
	routes.RegisterAuth(api, db)

	port := utils.GetEnv("AUTH_PORT", "8080")
	log.Printf("[auth] listening on :%s", port)
	log.Fatal(router.Run(":" + port))
}
