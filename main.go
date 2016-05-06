package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/contrib/static"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"os"
	"fmt"
)

func main() {
	router := gin.Default()
	router.NoRoute(static.ServeRoot("/", "./public")) //serve static files for other routes
	router.Use(catapultMiddleware) // make CatapultAPI available for all routes
	connectionString := os.Getenv("DATABASE_URI")
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
