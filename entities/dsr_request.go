package entities

import "time"

// DSRRequest is the audit record for every call to the Data Subject
// Rights API (delete/update/access), regardless of outcome.
type DSRRequest struct {
	ID                   string     `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	RequestType          string     `json:"request_type" gorm:"type:text;not null"`
	RequesterFirstName   string     `json:"requester_first_name" gorm:"type:text;not null"`
	RequesterLastName    string     `json:"requester_last_name" gorm:"type:text;not null"`
	RequesterEmail       *string    `json:"requester_email,omitempty" gorm:"type:text"`
	RequesterPhoneNumber string     `json:"requester_phone_number" gorm:"type:text;not null"`
	MatchedUserID        *string    `json:"matched_user_id,omitempty" gorm:"type:uuid"`
	Status               string     `json:"status" gorm:"type:text;not null"`
	Details              *string    `json:"details,omitempty" gorm:"type:jsonb"`
	ErrorMessage         *string    `json:"error_message,omitempty" gorm:"type:text"`
	RequestedOn          time.Time  `json:"requested_on" gorm:"type:timestamptz;not null;default:now()"`
	CompletedOn          *time.Time `json:"completed_on,omitempty" gorm:"type:timestamptz"`
}

func (DSRRequest) TableName() string {
	return "dsr_requests"
}
