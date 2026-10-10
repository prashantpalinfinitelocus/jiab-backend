package entities

import "time"

type Meme struct {
	ID         string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4();column:id"`
	UserID     string    `gorm:"type:uuid;not null;column:user_id"`
	Status     int       `gorm:"type:int;not null;default:1;column:status"`
	LanguageID int       `gorm:"type:int;not null;column:language_id"`
	TemplateID string    `gorm:"type:varchar(255);not null;column:template_id"`
	AssetURL   string    `gorm:"type:text;not null;column:asset_url"`
	ViewCount  int       `gorm:"type:int;not null;default:0;column:view_count"`
	CreatedAt  time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;column:created_at"`
	UpdatedAt  time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;column:updated_at"`
}

func (Meme) TableName() string { return "memes" }

type MemeReaction struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4();column:id"`
	MemeID    *string   `gorm:"type:uuid;column:meme_id"`
	JokeID    *string   `gorm:"type:uuid;column:joke_id"`
	UserID    string    `gorm:"type:uuid;not null;column:user_id"`
	Reaction  string    `gorm:"type:text;not null;column:reaction"`
	CreatedAt time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;column:created_at"`
}

func (MemeReaction) TableName() string { return "meme_reactions" }
