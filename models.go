package main

import (
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
	"fmt"
)

// MinPasswordLength defines minimal length of password
const MinPasswordLength = 6

const salt = "cWWRcK0.8^eUgu_!V@@K6D^;#,jL+Yl"

// User model
type User struct {
	gorm.Model
	ID           uint64
	UserName     string `gorm:"type:varchar(64);not null;unique_index"`
	PasswordHash []byte
	PhoneNumber  string
	SIPURI       string
	SIPPassword  string
}

// SetPassword set hash for password
func (u *User) SetPassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("Password length should be more or equal %d symbols", MinPasswordLength)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password+salt), bcrypt.DefaultCost)
	u.PasswordHash = hash
	return err
}

// ComparePasswords caompares hashed password with passed one
func (u *User) ComparePasswords(password string) bool {
	return bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password+salt)) == nil
}
