package services

import "testing"

func TestNormalizePhoneVariants(t *testing.T) {
	cases := []struct {
		name                string
		in                  string
		wantBare10          string
		wantWithCountryCode string
	}{
		{"bare 10 digit", "9569091401", "9569091401", "919569091401"},
		{"12 digit with 91 no plus", "919569091401", "9569091401", "919569091401"},
		{"E.164 with plus", "+919569091401", "9569091401", "919569091401"},
		{"unrecognized shape passed through both", "123", "123", "123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotBare10, gotWithCC := normalizePhoneVariants(tc.in)
			if gotBare10 != tc.wantBare10 || gotWithCC != tc.wantWithCountryCode {
				t.Errorf("normalizePhoneVariants(%q) = (%q, %q), want (%q, %q)",
					tc.in, gotBare10, gotWithCC, tc.wantBare10, tc.wantWithCountryCode)
			}
		})
	}
}

func TestValidateUpdateFields(t *testing.T) {
	cases := []struct {
		name    string
		fields  map[string]interface{}
		wantErr bool
	}{
		{"empty fields rejected", map[string]interface{}{}, true},
		{"all whitelisted fields accepted", map[string]interface{}{
			"name": "Jane", "email": "jane@example.com", "phone_number": "9569091401",
			"dob": "1990-01-01", "gender": "1", "alternate_mobile": "9000000000",
			"profile_picture": "https://example.com/p.png", "address": map[string]interface{}{},
		}, false},
		{"unknown field rejected", map[string]interface{}{"wallet_balance": "1000"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUpdateFields(tc.fields)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateUpdateFields(%v) error = %v, wantErr %v", tc.fields, err, tc.wantErr)
			}
		})
	}
}
