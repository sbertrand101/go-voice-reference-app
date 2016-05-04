package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func main() {
	router := gin.Default()
	db, err := gorm.Open("postgres", "GoLangVoiceReferenceApp")
	if err != nil {
		panic("Failed to connect database")
	}
	getRoutes(router, db)
	router.Run()
}
