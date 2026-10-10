package services

import (
	"fmt"
	"strings"

	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
)

// normalizePhoneVariants returns the two shapes a phone number might be
// stored as in users.phone_number: bare 10-digit, and 91-prefixed 12-digit
// (no "+"). Given any reasonable input (bare 10-digit, 91-prefixed 12-digit,
// or E.164 "+91..."), both return values let a caller match either stored
// shape with `WHERE phone_number = ? OR phone_number = ?`.
func normalizePhoneVariants(raw string) (bare10, withCountryCode string) {
	digits := strings.TrimSpace(raw)
	digits = strings.TrimPrefix(digits, "+")

	switch {
	case len(digits) == 10:
		return digits, "91" + digits
	case len(digits) == 12 && strings.HasPrefix(digits, "91"):
		return digits[2:], digits
	default:
		return digits, digits
	}
}

var dsrUpdatableFields = map[string]struct{}{
	"name":             {},
	"email":            {},
	"phone_number":     {},
	"dob":              {},
	"gender":           {},
	"alternate_mobile": {},
	"profile_picture":  {},
	"address":          {},
}

// validateUpdateFields rejects any key outside the DSR-updatable whitelist,
// and rejects an empty fields map (an update request must update something).
func validateUpdateFields(fields map[string]interface{}) error {
	if len(fields) == 0 {
		return errs.NewBadRequestError("fields is required and must not be empty for request_type=update", nil)
	}
	for key := range fields {
		if _, ok := dsrUpdatableFields[key]; !ok {
			return errs.NewBadRequestError(fmt.Sprintf("field %q is not updatable via this API", key), nil)
		}
	}
	return nil
}
