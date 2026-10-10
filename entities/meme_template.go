package entities

import "time"

// MemeTemplate is an internally-hosted meme GIF template (GCS-backed),
// replacing the previous Giphy-sourced template catalog.
type MemeTemplate struct {
	ID               string     `gorm:"primaryKey;type:uuid;default:uuid_generate_v4();column:id"`
	Episode          string     `gorm:"type:varchar(50);column:episode"`
	Genre            string     `gorm:"type:varchar(50);not null;column:genre"`
	GCSObjectPath    string     `gorm:"type:text;not null;column:gcs_object_path"`
	GCSURL           string     `gorm:"type:text;not null;column:gcs_url"`
	OriginalFilename string     `gorm:"type:varchar(255);column:original_filename"`
	ContentType      string     `gorm:"type:varchar(100);column:content_type"`
	IsActive         bool       `gorm:"type:bool;not null;default:true;column:is_active"`
	IsDeleted        bool       `gorm:"type:bool;not null;default:false;column:is_deleted"`
	CreatedOn        time.Time  `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;column:created_on"`
	LastModifiedOn   *time.Time `gorm:"type:timestamp;column:last_modified_on"`
}

func (MemeTemplate) TableName() string { return "meme_templates" }
