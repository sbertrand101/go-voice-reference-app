package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"os"
)

func main() {
	router := gin.Default()
	databaseSource := os.Getenv("DATABASE")
	if databaseSource == "" {
		databaseSource = "GoLangVoiceReferenceApp"
	}
	db, err := gorm.Open("postgres", databaseSource)
	if err != nil {
		panic("Failed to connect database")
	}
	getRoutes(router, db)
	router.Run()
}
