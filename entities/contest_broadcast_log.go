package entities

import "time"

// ContestBroadcastLog records that a contest announcement broadcast (launch or
// reminder) has been sent for a given contest, so it is never sent twice.
type ContestBroadcastLog struct {
	ID               int64     `gorm:"column:id;primaryKey;autoIncrement"`
	ContestID        string    `gorm:"column:contest_id;not null;uniqueIndex:idx_contest_broadcast_unique"`
	AnnouncementType string    `gorm:"column:announcement_type;not null;uniqueIndex:idx_contest_broadcast_unique"`
	RecipientCount   int       `gorm:"column:recipient_count;not null"`
	CreatedOn        time.Time `gorm:"column:created_on;not null"`
}

// TableName specifies the table name for GORM
func (ContestBroadcastLog) TableName() string {
	return "contest_broadcast_log"
}
