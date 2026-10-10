package dtos

import "time"

type RegisterDeviceDTO struct {
	DeviceToken string `json:"device_token" binding:"required"`
}

type NotificationResponseDTO struct {
	NotificationTitle string     `json:"notification_title"`
	NotificationText  string     `json:"notification_text"`
	IsNew             bool       `json:"is_new"`
	IsRead            bool       `json:"is_read"`
	IconURL           *string    `json:"icon_url"`
	LaunchDate        *time.Time `json:"launch_date"`
}

type NotificationFilter struct {
	PageNumber int `json:"page_number"`
	PageSize   int `json:"page_size"`
}

type NotificationResponse struct {
	Notifications []NotificationResponseDTO `json:"notifications"`
	TotalCount    int                       `json:"total_count"`
	PageNumber    int                       `json:"page_number,omitempty"`
	PageSize      int                       `json:"page_size,omitempty"`
}

type NotificationCountResponse struct {
	Count int `json:"count"`
}

type KafkaMessage struct {
	Token    string `json:"token"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	ImageURL string `json:"image_url"`
}

type PubSubMessage struct {
	Token    string `json:"token"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	ImageURL string `json:"image_url"`
}

type PublishMessageDTO struct {
	Topic   string        `json:"topic" binding:"required"`
	Key     string        `json:"key,omitempty"`
	Message PubSubMessage `json:"message" binding:"required"`
}

type PublishResponseDTO struct {
	MessageID string `json:"message_id"`
	Success   bool   `json:"success"`
}

type BroadcastNotificationDTO struct {
	Title    string `json:"title" binding:"required"`
	Body     string `json:"body" binding:"required"`
	ImageURL string `json:"image_url,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
}

type PhoneNotificationDTO struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Body        string `json:"body" binding:"required"`
	ImageURL    string `json:"image_url,omitempty"`
}

type FCMDeviceTokenResponseDTO struct {
	ID             int64      `json:"id"`
	UserID         string     `json:"user_id"`
	DeviceToken    string     `json:"device_token"`
	IsActive       bool       `json:"is_active"`
	IsDeleted      bool       `json:"is_deleted"`
	CreatedOn      time.Time  `json:"created_on"`
	LastModifiedOn *time.Time `json:"last_modified_on"`
}

type GetTokensByDateResponseDTO struct {
	Tokens []FCMDeviceTokenResponseDTO `json:"tokens"`
	Count  int                         `json:"count"`
	Date   string                      `json:"date"`
}

// NotificationSendResult summarizes the outcome of a batch/campaign notification run.
type NotificationSendResult struct {
	Eligible int `json:"eligible"`
	Sent     int `json:"sent"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
}

// ProfileCompletionCampaignDTO is the request body for the ops-triggered
// profile-completion bulk campaign (with a coin reward attached).
type ProfileCompletionCampaignDTO struct {
	Title    string `json:"title" binding:"required"`
	Body     string `json:"body" binding:"required"`
	ImageURL string `json:"image_url,omitempty"`
}

// ContestAnnouncementDTO is the request body for the contest announcement broadcast.
type ContestAnnouncementDTO struct {
	ContestID        string `json:"contest_id" binding:"required"`
	AnnouncementType string `json:"announcement_type" binding:"required"` // "launch" or "reminder"
	Title            string `json:"title" binding:"required"`
	Body             string `json:"body" binding:"required"`
	ImageURL         string `json:"image_url,omitempty"`
}
