package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"os"
	"github.com/jinzhu/gorm"
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

func TestAutoMigrate(t *testing.T) {
	connectionString := os.Getenv("DATABASE_URI")
	if connectionString == "" {
		connectionString = "postgresql://postgres@localhost/golang_voice_reference_app_test?sslmode=disable"
	}
	db, err := gorm.Open("postgres", connectionString)
	assert.NoError(t, err)
	db.DropTableIfExists(&User{})
	assert.NoError(t, AutoMigrate(db).Error)
	assert.True(t, db.HasTable(&User{}))
}
