package entities

type CDPEventPayload struct {
	ClientID                 string                 `json:"client_id"`
	EventType                string                 `json:"event_type"`
	EventSubType             string                 `json:"event_sub_type"`
	BrandName                string                 `json:"brand_name"`
	PhoneE164                string                 `json:"phone_e164"`
	Email                    *string                `json:"email,omitempty"`
	FirstName                *string                `json:"first_name,omitempty"`
	TargetingAgeFrom         *string                `json:"targeting_age_from,omitempty"`
	CommunicationPreferences map[string]int         `json:"communication_preferences,omitempty"`
	GeoCountryCode           *string                `json:"geo_country_code,omitempty"`
	GeoStateProvinceCode     *string                `json:"geo_state_province_code,omitempty"`
	GeoCityName              *string                `json:"geo_city_name,omitempty"`
	GeoPostalCode            *string                `json:"geo_postal_code,omitempty"`
	IPAddress                *string                `json:"ip_address,omitempty"`
	Referrer                 *string                `json:"referrer,omitempty"`
	LanguageCode             *string                `json:"language_code,omitempty"`
	AddressLine1             *string                `json:"address_line1,omitempty"`
	AddressLine2             *string                `json:"address_line2,omitempty"`
	AddressCity              *string                `json:"address_city,omitempty"`
	AddressState             *string                `json:"address_state,omitempty"`
	DOB                      *string                `json:"dob,omitempty"`
	Gender                   *string                `json:"gender,omitempty"`
	AdditionalData           map[string]interface{} `json:"additional_data,omitempty"`
}
