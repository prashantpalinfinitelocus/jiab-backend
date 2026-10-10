package entities

import "time"

type MemeViewHistory struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4();column:id"`
	UserID    string    `gorm:"type:uuid;not null;column:user_id"`
	MemeID    string    `gorm:"type:uuid;not null;column:meme_id"`
	ViewedAt  time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;column:viewed_at"`
	CreatedAt time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;column:created_at"`
}

func (MemeViewHistory) TableName() string {
	return "meme_view_history"
}
