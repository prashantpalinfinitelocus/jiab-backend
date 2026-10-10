package entities

import "time"

type JokeFormat struct {
	ID             int        `gorm:"column:id;primaryKey"`
	Value          string     `gorm:"column:value;not null"`
	Title          string     `gorm:"column:title;not null"`
	IconURL        string     `gorm:"column:icon_url;not null"`
	IsActive       bool       `gorm:"column:is_active;not null"`
	IsDeleted      bool       `gorm:"column:is_deleted;not null"`
	CreatedBy      string     `gorm:"column:created_by;not null"`
	CreatedOn      time.Time  `gorm:"column:created_on;not null"`
	LastModifiedBy *string    `gorm:"column:last_modified_by"`
	LastModifiedOn *time.Time `gorm:"column:last_modified_on"`
}

func (JokeFormat) TableName() string {
	return "joke_formats"
}
