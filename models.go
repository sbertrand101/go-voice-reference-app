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
	ID           uint64 `gorm:"primary_key"`
	UserName     string `gorm:"type:varchar(64);not null;unique_index"`
	PasswordHash []byte
	PhoneNumber  string `gorm:"type:varchar(32);unique_index"`
	SIPURI       string `gorm:"column:sip_uri;type:varchar(1024)"`
	SIPPassword  string `gorm:"column:sip_password;type:varchar(128)"`
}

// SetPassword sets hash for password
func (u *User) SetPassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("Password length should be more or equal %d symbols", MinPasswordLength)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password+salt), bcrypt.DefaultCost)
	u.PasswordHash = hash
	return err
}

// ComparePasswords compares hashed password with parameter
func (u *User) ComparePasswords(password string) bool {
	return bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password+salt)) == nil
}

// AutoMigrate updates tables in db using models definitions
func AutoMigrate(db *gorm.DB) *gorm.DB {
	return db.AutoMigrate(&User{})
}
