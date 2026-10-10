package entities

import (
	"time"
)

type UserKOCID struct {
	ID          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      string    `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
	HashedKOCID string    `json:"hashed_kocid" gorm:"type:varchar(255);not null"`
	SyncedAt    time.Time `json:"synced_at" gorm:"type:timestamp;not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"type:timestamp;not null;default:now()"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"type:timestamp;not null;default:now()"`

	User User `json:"user" gorm:"foreignKey:UserID;references:ID"`
}

func (UserKOCID) TableName() string {
	return "user_kocid"
}
