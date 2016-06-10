package main

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

// MinPasswordLength defines minimal length of password
const MinPasswordLength = 6

const salt = "cWWRcK0.8^eUgu_!V@@K6D^;#,jL+Yl"

// User model
type User struct {
	gorm.Model
	UserName          string `gorm:"type:varchar(64);not null;unique_index"`
	PasswordHash      []byte
	AreaCode          string `gorm:"type:char(3)"`
	PhoneNumber       string `gorm:"type:varchar(32);unique_index"`
	EndpointID        string `gorm:"column:endpoint_id;type:varchar(64)"`
	SIPURI            string `gorm:"column:sip_uri;type:varchar(1024);index"`
	SIPPassword       string `gorm:"column:sip_password;type:varchar(128)"`
	GreetingURL       string `gorm:"column:greeting_url;type:varchar(1024)"`
	VoiceMailMessages []VoiceMailMessage
}

// VoiceMailMessage model
type VoiceMailMessage struct {
	gorm.Model
	User      User `gorm:"ForeignKey:UserID"`
	UserID    uint
	StartTime time.Time `gorm:"index"`
	EndTime   time.Time
	MediaURL  string `gorm:"column:media_url;type:varchar(1024)"`
	From      string
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

// ToJSONObject returns map presentation of model instance (usefull for json)
func (m *VoiceMailMessage) ToJSONObject() map[string]interface{} {
	return map[string]interface{}{
		"startTime": m.StartTime,
		"endTime":   m.EndTime,
		"from":      m.From,
		"id":        m.ID,
	}
}

// ActiveCall model
type ActiveCall struct {
	CreatedAt time.Time `gorm:"index"`
	UserID    uint      `gorm:"column:user_id;not_null;index"`
	CallID    string    `gorm:"column:call_id;type:varchar(64);not_null;index"`
	BridgeID  string    `gorm:"column:bridge_id;type:varchar(64);index"`
	From      string
	To        string
}

// AutoMigrate updates tables in db using models definitions
func AutoMigrate(db *gorm.DB) *gorm.DB {
	execSQL := !db.HasTable(&ActiveCall{})
	db.AutoMigrate(&User{}, &VoiceMailMessage{}, &ActiveCall{})
	// Postgresql will remove expired records itself
	if execSQL {
		time.Sleep(2 * time.Second)
		db.Exec(`CREATE OR REPLACE FUNCTION delete_old_rows()
		RETURNS trigger AS
		$BODY$
		BEGIN
		DELETE FROM active_calls WHERE created_at < NOW() - INTERVAL '2 hours';
		RETURN NULL;
		END;
		$BODY$
		LANGUAGE plpgsql VOLATILE
		COST 100;`)
		db.Exec(`CREATE TRIGGER "RemoveExpiredRecords"
		AFTER INSERT
		ON active_calls
		FOR EACH ROW
		EXECUTE PROCEDURE delete_old_rows();`)
	}
	return db
}
