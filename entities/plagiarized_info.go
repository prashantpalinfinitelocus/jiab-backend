package entities

import "time"

type PlagiarizedInfo struct {
	ID             int        `json:"id" gorm:"primaryKey;autoIncrement"`
	AssetID        string     `json:"asset_id" gorm:"type:uuid;not null"`
	UserID         string     `json:"user_id" gorm:"type:uuid;not null"`
	Information    string     `json:"information" gorm:"type:text;not null"`
	IsActive       bool       `json:"is_active" gorm:"type:boolean;not null;default:true"`
	IsDeleted      bool       `json:"is_deleted" gorm:"type:boolean;not null;default:false"`
	CreatedBy      string     `json:"created_by" gorm:"type:text;not null"`
	CreatedOn      time.Time  `json:"created_on" gorm:"type:timestamp;not null;default:now()"`
	LastModifiedBy *string    `json:"last_modified_by" gorm:"type:text"`
	LastModifiedOn *time.Time `json:"last_modified_on" gorm:"type:timestamp"`
}

func (PlagiarizedInfo) TableName() string {
	return "plagiarized_infos"
}
