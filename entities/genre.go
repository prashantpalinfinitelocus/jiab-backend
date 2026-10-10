package entities

import (
	"time"
)

type JokeGenre struct {
	ID             int        `gorm:"primaryKey;column:id"`
	Genre          string     `gorm:"column:genre;not null"`
	GenreEmoticon  string     `gorm:"column:genre_emoticon;not null"`
	ImageURL       string     `gorm:"column:image_url;not null"`
	EmoIsActive    bool       `gorm:"column:emo_is_active;not null"`
	EmoIndex       int        `gorm:"column:emo_index;not null"`
	IsActive       bool       `gorm:"column:is_active;not null"`
	IsDeleted      bool       `gorm:"column:is_deleted;not null"`
	CreatedBy      string     `gorm:"column:created_by;not null"`
	CreatedOn      time.Time  `gorm:"column:created_on;not null"`
	LastModifiedBy *string    `gorm:"column:last_modified_by"`
	LastModifiedOn *time.Time `gorm:"column:last_modified_on"`
}

func (JokeGenre) TableName() string {
	return "joke_genres"
}
