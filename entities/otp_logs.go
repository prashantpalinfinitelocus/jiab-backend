package entities

import (
	"time"
)

type OTPLogs struct {
	ID             int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	MobileNo       string     `json:"mobile_no" gorm:"type:varchar(15);not null"`
	OTP            string     `json:"otp" gorm:"type:varchar(6);not null"`
	Attempts       int32      `json:"attempts" gorm:"type:int;not null;default:0"`
	IsActive       bool       `json:"is_active" gorm:"type:boolean;not null;default:true"`
	IsDeleted      bool       `json:"is_deleted" gorm:"type:boolean;not null;default:false"`
	CreatedBy      string     `json:"created_by" gorm:"type:varchar(255);not null"`
	CreatedOn      time.Time  `json:"created_on" gorm:"type:timestamp;not null"`
	LastModifiedOn *time.Time `json:"last_modified_on,omitempty" gorm:"type:timestamp"`
	LastModifiedBy string     `json:"last_modified_by,omitempty" gorm:"type:varchar(255);not null"`
}

func (OTPLogs) TableName() string {
	return "otp_logs"
}
