package entities

import (
	"time"
)

type UserCreatorCollabReaction struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	UserID       string    `json:"user_id" gorm:"type:uuid;not null;column:user_id"`
	JokeID       string    `json:"joke_id" gorm:"type:uuid;not null;column:joke_id"`
	Reaction     string    `json:"reaction" gorm:"type:text;not null;column:reaction"`
	ReactionDate time.Time `json:"reaction_date" gorm:"type:date;not null;column:reaction_date"`
	ReactionDttm time.Time `json:"reaction_dttm" gorm:"type:timestamptz;not null;column:reaction_dttm"`
}

func (UserCreatorCollabReaction) TableName() string {
	return "ad_user_creator_collab_reaction"
}
