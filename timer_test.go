package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSleep(t *testing.T) {
	api := &timer{}
	api.Sleep(0)
}

func TestTimerMiddleware(t *testing.T) {
	context := createFakeGinContext()
	timerMiddleware(context)
	_, ok := context.Get("timerAPI")
	assert.True(t, ok)
}
