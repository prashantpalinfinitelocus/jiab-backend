package entities

import "time"

type MemeText struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4();column:id"`
	MemeID    string    `gorm:"type:uuid;not null;column:meme_id;index"`
	Text1     *string   `gorm:"type:text;column:text1"`
	Text2     *string   `gorm:"type:text;column:text2"`
	CreatedAt time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;column:created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;column:updated_at"`
}

func (MemeText) TableName() string { return "meme_texts" }
