package entities

import "time"

type NotificationType struct {
	ID               int        `gorm:"column:id;primaryKey;autoIncrement"`
	Name             string     `gorm:"column:name;not null"`
	TitleValue       string     `gorm:"column:title_value;not null"`
	IconURL          *string    `gorm:"column:icon_url"`
	LanguageMasterID int        `gorm:"column:language_master_id;not null"`
	IsBroadcast      bool       `gorm:"column:is_broadcast;not null"`
	IsActive         bool       `gorm:"column:is_active;not null"`
	IsDeleted        bool       `gorm:"column:is_deleted;not null"`
	CreatedBy        string     `gorm:"column:created_by;not null"`
	CreatedOn        time.Time  `gorm:"column:created_on;not null"`
	LastModifiedBy   *string    `gorm:"column:last_modified_by"`
	LastModifiedOn   *time.Time `gorm:"column:last_modified_on"`
}

// TableName specifies the table name for GORM
func (NotificationType) TableName() string {
	return "notification_types"
}

type UserNotification struct {
	ID                 int64      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID             string     `gorm:"column:user_id;not null"`
	NotificationTypeID int        `gorm:"column:notification_type_id;not null"`
	NotificationTitle  string     `gorm:"column:notification_title;not null"`
	NotificationText   string     `gorm:"column:notification_text;not null"`
	IsNew              bool       `gorm:"column:is_new;not null"`
	IsRead             bool       `gorm:"column:is_read;not null"`
	LaunchDate         *time.Time `gorm:"column:launch_date"`
	IsActive           bool       `gorm:"column:is_active;not null"`
	IsDeleted          bool       `gorm:"column:is_deleted;not null"`
	CreatedBy          string     `gorm:"column:created_by;not null"`
	CreatedOn          time.Time  `gorm:"column:created_on;not null"`
	LastModifiedBy     *string    `gorm:"column:last_modified_by"`
	LastModifiedOn     *time.Time `gorm:"column:last_modified_on"`

	User             User             `gorm:"foreignKey:UserID;references:ID"`
	NotificationType NotificationType `gorm:"foreignKey:NotificationTypeID;references:ID"`
}

// TableName specifies the table name for GORM
func (UserNotification) TableName() string {
	return "user_notification"
}

type FCMDeviceToken struct {
	ID             int64      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID         string     `gorm:"column:user_id;not null"`
	DeviceToken    string     `gorm:"column:device_token;not null"`
	IsActive       bool       `gorm:"column:is_active;not null"`
	IsDeleted      bool       `gorm:"column:is_deleted;not null"`
	CreatedOn      time.Time  `gorm:"column:created_on;not null"`
	LastModifiedOn *time.Time `gorm:"column:last_modified_on"`

	User User `gorm:"foreignKey:UserID;references:ID"`
}

func (FCMDeviceToken) TableName() string {
	return "fcm_device_tokens"
}
