package entities

import "time"

type WAAssetActivity struct {
	ID             int        `json:"id" gorm:"primaryKey;autoIncrement"`
	AssetID        string     `json:"asset_id" gorm:"not null"`
	ActivityType   int        `json:"activity_type" gorm:"type:int;not null"`
	CreatedOn      time.Time  `json:"created_on" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	LastModifiedOn *time.Time `json:"last_modified_on" gorm:"type:timestamp"`
}

func (WAAssetActivity) TableName() string {
	return "wa_asset_activity"
}
