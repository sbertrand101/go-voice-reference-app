package main

import (
	"time"

	"github.com/gin-gonic/gin"
)

type timer struct{}

type timerInterface interface {
	Sleep(d time.Duration)
}

func (t *timer) Sleep(d time.Duration) {
	time.Sleep(d)
}

func timerMiddleware(c *gin.Context) {
	c.Set("timerAPI", &timer{})
	c.Next()
}
