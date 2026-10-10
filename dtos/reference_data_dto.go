package dtos

type ReferenceDataDTO struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ImageURL *string `json:"imageUrl,omitempty"`
}
