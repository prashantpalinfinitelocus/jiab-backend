package dtos

type LocationResponseDTO struct {
	Pincode int    `json:"pincode"`
	City    string `json:"city"`
	State   string `json:"state"`
}
