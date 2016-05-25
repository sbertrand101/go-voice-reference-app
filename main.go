package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func main() {
	router := gin.Default()
	router.NoRoute(static.ServeRoot("/", "./public")) //serve static files for other routes
	router.Use(catapultMiddleware)                    // make CatapultAPI available for all routes
	router.Use(timerMiddleware)
	connectionString := os.Getenv("DATABASE_URL")
	if connectionString == "" {
		// Docker's links support
		host := os.Getenv("DB_PORT_5432_TCP_ADDR")
		port := os.Getenv("DB_PORT_5432_TCP_PORT")
		if host != "" && port != "" {
			connectionString = fmt.Sprintf("postgresql://postgres@%s:%s/postgres?sslmode=disable", host, port)
		}
	}
	if connectionString == "" {
		connectionString = "postgresql://postgres@localhost/golang_voice_reference_app?sslmode=disable"
	}
	db, err := gorm.Open("postgres", connectionString)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect database: %s", err.Error()))
	}
	if err = AutoMigrate(db).Error; err != nil {
		panic(fmt.Sprintf("Error on executing db migrations: %s", err.Error()))
	}
	if err = getRoutes(router, db); err != nil {
		panic(fmt.Sprintf("Error on creating routes: %s", err.Error()))
	}
	router.Run()
}
