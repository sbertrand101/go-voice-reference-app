package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUserSetPassword(t *testing.T) {
	user := &User{}
	assert.True(t, len(user.PasswordHash) == 0)
	assert.NoError(t, user.SetPassword("123456"))
	assert.True(t, len(user.PasswordHash) > 0)
}

func TestUserSetShortPassword(t *testing.T) {
	user := &User{}
	assert.True(t, len(user.PasswordHash) == 0)
	assert.Error(t, user.SetPassword("123"))
	assert.True(t, len(user.PasswordHash) == 0)
}

func TestUserComparePasswords(t *testing.T) {
	user := &User{}
	assert.NoError(t, user.SetPassword("123456"))
	assert.True(t, user.ComparePasswords("123456"))
	assert.False(t, user.ComparePasswords("1234567"))
}

func TestVoiceMailMessageToJSONObject(t *testing.T) {
	message := &VoiceMailMessage{
		StartTime: time.Now(),
		EndTime:   time.Now(),
	}
	message.ID = 1
	result := message.ToJSONObject()
	assert.NotEmpty(t, result["startTime"])
	assert.NotEmpty(t, result["endTime"])
	assert.NotEmpty(t, result["id"])
}

func TestAutoMigrate(t *testing.T) {
	db := openDBConnection(t)
	assert.NoError(t, AutoMigrate(db).Error)
	assert.True(t, db.HasTable(&User{}))
}
