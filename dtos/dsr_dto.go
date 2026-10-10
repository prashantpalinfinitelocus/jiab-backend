package dtos

// DSRRequestDTO is the request body for POST /dsr/requests, called by
// OneTrust's DSR automation to fulfill a privacy-portal request.
type DSRRequestDTO struct {
	RequestType string                 `json:"request_type" binding:"required,oneof=delete update access"`
	FirstName   string                 `json:"first_name,omitempty"`
	LastName    string                 `json:"last_name,omitempty"`
	Email       string                 `json:"email,omitempty"`
	PhoneNumber string                 `json:"phone_number" binding:"required"`
	Country     string                 `json:"country,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
}

// DSRAddressFieldDTO is the shape expected when fields["address"] is
// present on an update request. The caller supplies only the pincode value
// (not the internal pin_code/city/state row IDs, which an external privacy
// portal has no way of knowing) — the service resolves pin_code_id, city_id,
// and state_id from the pincode via a lookup.
type DSRAddressFieldDTO struct {
	Address1        string  `json:"address1" binding:"required"`
	Address2        *string `json:"address2,omitempty"`
	Pincode         int     `json:"pincode" binding:"required"`
	NearestLandmark *string `json:"nearest_landmark,omitempty"`
	ShippingMobile  *string `json:"shipping_mobile,omitempty"`
}

// DSRResponseDTO is the `data` payload returned for every request_type.
type DSRResponseDTO struct {
	RequestID        string      `json:"request_id"`
	Status           string      `json:"status"`
	AnonymizedTables []string    `json:"anonymized_tables,omitempty"`
	UpdatedFields    []string    `json:"updated_fields,omitempty"`
	ExportData       interface{} `json:"export_data,omitempty"`
}
