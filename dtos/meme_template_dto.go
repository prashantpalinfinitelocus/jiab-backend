package dtos

// BulkRegisterMemeTemplateItem describes one already-uploaded GCS object to
// register as a meme template. Upload and registration are deliberately
// separate steps - this just records metadata for files already in the
// bucket.
type BulkRegisterMemeTemplateItem struct {
	GCSObjectPath    string `json:"gcs_object_path" binding:"required"`
	GCSURL           string `json:"gcs_url" binding:"required"`
	Episode          string `json:"episode"`
	Genre            string `json:"genre" binding:"required"`
	OriginalFilename string `json:"original_filename"`
	ContentType      string `json:"content_type"`
}

type BulkRegisterMemeTemplatesRequest struct {
	Templates []BulkRegisterMemeTemplateItem `json:"templates" binding:"required,dive"`
}

type BulkRegisterMemeTemplatesResponse struct {
	Registered int `json:"registered"`
}
