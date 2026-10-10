package entities

import "time"

type UserRank struct {
	ID             int        `gorm:"column:id;primaryKey;autoIncrement"`
	UserRank       int64      `gorm:"column:user_rank;not null"`
	UserID         string     `gorm:"column:user_id;not null;type:varchar(450)"`
	CurrentBalance *int       `gorm:"column:current_balance"`
	Lock           bool       `gorm:"column:lock;not null"`
	LastModified   *time.Time `gorm:"column:last_modified"`
	RowVersion     time.Time  `gorm:"column:rowversion;not null"`
	SysStartTime   time.Time  `gorm:"column:sys_start_time;not null"`
	SysEndTime     time.Time  `gorm:"column:sys_end_time;not null"`
}

// TableName specifies the table name for UserRank
func (UserRank) TableName() string {
	return "user_ranks"
}
