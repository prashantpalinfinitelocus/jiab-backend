package entities

import "time"

type MemeVoteHistory struct {
	ID       int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID   string    `json:"user_id" gorm:"type:text"`
	AssetID  string    `json:"asset_id" gorm:"type:text"`
	Language string    `json:"language" gorm:"type:varchar(50)"`
	VotedAt  time.Time `json:"voted_at" gorm:"type:timestamp"`
}

func (MemeVoteHistory) TableName() string {
	return "meme_vote_history"
}
