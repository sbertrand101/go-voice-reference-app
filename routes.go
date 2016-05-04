package main

import (
	"strconv"
	"time"

	"errors"
	"net/http"

	"github.com/appleboy/gin-jwt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// RegisterForm is used on used registering
type RegisterForm struct {
	User           string `form:"user",json:"user",binding:"required"`
	Password       string `form:"password",json:"password",binding:"required"`
	RepeatPassword string `form:"repeatPassword",json:"repeatPassword",binding:"required"`
}

func getRoutes(router *gin.Engine, db *gorm.DB) {
	authMiddleware := &jwt.GinJWTMiddleware{
		Realm:      "Bandwidth",
		Key:        []byte("9SbPxeIyvoT3HkIQ19wN9p_e_b6Xb7iJ"),
		Timeout:    time.Hour,
		MaxRefresh: time.Hour * 24,
		Authenticator: func(userId string, password string, c *gin.Context) (string, bool) {
			user := &User{}
			if db.First(user, "UserName = ?", userId).RecordNotFound() {
				return "", false
			}
			return strconv.FormatUint(user.ID, 10), user.ComparePasswords(password)
		},
		Authorizator: func(userId string, c *gin.Context) bool {
			user := &User{}
			id, err := strconv.ParseUint(userId, 10, 64)
			if err != nil {
				return false
			}
			if db.First(user, id).RecordNotFound() {
				return false
			}
			c.Set("user", user)
			return true
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		},
	}

	router.POST("/login", authMiddleware.LoginHandler)
	router.GET("/refreshToken", authMiddleware.MiddlewareFunc(), authMiddleware.RefreshHandler)

	router.POST("/register", func(c *gin.Context) {
		user := &User{}
		form := &RegisterForm{}
		err := c.Bind(form)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		if form.Password != form.RepeatPassword {
			c.AbortWithError(http.StatusBadRequest, errors.New("Passwords are mismatched"))
			return
		}
		if err = db.Create(user).Error; err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}
		c.JSON(http.StatusOK, gin.H{
			"id": user.ID,
		})
	})

	router.GET("/", authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"private": "data",
		})
	})
}
