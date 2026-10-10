package entities

import "time"

type VoteHistory struct {
	ID       int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID   string    `json:"user_id" gorm:"type:text"`
	Genre    string    `json:"genre" gorm:"type:varchar(50)"`
	Language string    `json:"language" gorm:"type:varchar(50)"`
	VotedAt  time.Time `json:"voted_at" gorm:"type:timestamp"`
}

func (VoteHistory) TableName() string {
	return "user_vote_history"
}
