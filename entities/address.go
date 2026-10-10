package entities

import (
	"time"
)

type Address struct {
	ID              int        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID          string     `json:"user_id" gorm:"type:text;not null"`
	Address1        string     `json:"address1" gorm:"type:text;not null"`
	Address2        *string    `json:"address2" gorm:"type:text"`
	Pincode         int        `json:"pincode" gorm:"type:int;not null"`
	PinCodeID       int        `json:"pin_code_id" gorm:"type:int;not null"`
	CityID          int        `json:"city_id" gorm:"type:int;not null"`
	StateID         int        `json:"state_id" gorm:"type:int;not null"`
	NearestLandmark *string    `json:"nearest_landmark" gorm:"type:text"`
	ShippingMobile  *string    `json:"shipping_mobile" gorm:"type:text"`
	IsDefault       bool       `json:"is_default" gorm:"type:bool;not null"`
	IsActive        bool       `json:"is_active" gorm:"type:bool;not null"`
	IsDeleted       bool       `json:"is_deleted" gorm:"type:bool;not null"`
	CreatedBy       string     `json:"created_by" gorm:"type:text;not null"`
	CreatedOn       time.Time  `json:"created_on" gorm:"type:timestamp;not null"`
	LastModifiedBy  *string    `json:"last_modified_by" gorm:"type:text"`
	LastModifiedOn  *time.Time `json:"last_modified_on" gorm:"type:timestamp"`
}

func (Address) TableName() string {
	return "user_address"
}
