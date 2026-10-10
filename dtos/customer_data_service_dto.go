package dtos

// Request DTOs
type GetHashedKOCIDRequest struct {
	PhoneWithCountryCode string `json:"phone_with_countrycode,omitempty"`
	ClientID             string `json:"client_id"`
}

type UserIdentifier struct {
	UserIdentifier        string `json:"user_identifier"`
	UserIdentifierType    string `json:"user_identifier_type"`
	UserIdentifierSubType string `json:"user_identifier_sub_type"`
}

type DatamartCollectRequest struct {
	ClientID             string           `json:"client_id"`
	Email                string           `json:"email,omitempty"`
	PhoneWithCountryCode string           `json:"phone_with_countrycode,omitempty"`
	FirstName            string           `json:"first_name,omitempty"`
	LastName             string           `json:"last_name,omitempty"`
	DOB                  string           `json:"dob,omitempty"`
	Gender               string           `json:"gender,omitempty"`
	UserIdentifiers      []UserIdentifier `json:"user_identifiers,omitempty"`
}

// Response DTOs
type GetHashedKOCIDResponse struct {
	Status int              `json:"status"`
	Data   *HashedKOCIDData `json:"data,omitempty"`
	Error  string           `json:"error,omitempty"`
}

type HashedKOCIDData struct {
	KOIdentifier string `json:"ko_identifier"`
}

type DatamartCollectResponse struct {
	Status int    `json:"status"`
	Error  string `json:"error,omitempty"`
}

// API Response DTO
type UserKOCIDResponse struct {
	KOCID    *string `json:"kocid"`
	SyncedAt *string `json:"synced_at,omitempty"`
	Message  string  `json:"message,omitempty"`
}
