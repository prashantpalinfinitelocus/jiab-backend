package errors

import (
	"fmt"
)

// AppError represents an application-specific error.
type AppError struct {
	Code    int
	Message string
	Status  int
	Err     error
	Details map[string]interface{}
}

// NewAppError creates a new AppError with the given code, message, status, and an optional original error.
func NewAppError(code int, message string, status int, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
		Err:     err,
	}
}

// NewAppErrorWithDetails creates a new AppError with additional details.
func NewAppErrorWithDetails(code int, message string, status int, err error, details map[string]interface{}) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
		Err:     err,
		Details: details,
	}
}

// AddDetail adds a key-value pair to the error details.
func (e *AppError) AddDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// AddValidationError adds a validation error for a specific field.
func (e *AppError) AddValidationError(field, message string) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}

	// Initialize validation_errors if it doesn't exist
	if e.Details["validation_errors"] == nil {
		e.Details["validation_errors"] = make(map[string][]string)
	}

	validationErrors := e.Details["validation_errors"].(map[string][]string)
	validationErrors[field] = append(validationErrors[field], message)

	return e
}

// Error implements the error interface for AppError.
// It includes the original error's message if present.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("Error %d: %s (Caused by: %v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("Error %d: %s", e.Code, e.Message)
}

var (
	ErrNotFound                      = NewAppError(1001, "Resource not found", 404, nil)
	ErrBadRequest                    = NewAppError(1002, "Bad request", 400, nil)
	ErrInternalServer                = NewAppError(1003, "Internal server error", 500, nil)
	ErrUnauthorized                  = NewAppError(1004, "Unauthorized", 401, nil)
	ErrForbidden                     = NewAppError(1005, "Forbidden", 403, nil)
	ErrConflict                      = NewAppError(1006, "Conflict", 409, nil)
	ErrUnprocessableEntity           = NewAppError(1007, "Unprocessable entity", 422, nil)
	ErrCannotReferYourself           = NewAppError(1009, "You cannot refer yourself", 400, nil)
	ErrWrongReferralCode             = NewAppError(1010, "Wrong referral code", 400, nil)
	ErrReferralAlreadyExists         = NewAppError(1011, "An active referral already exists for this user", 409, nil)
	ErrTooManyRequests               = NewAppError(1012, "Too many requests", 429, nil)
	ErrMaximumJokeLimitReached       = NewAppError(1013, "Maximum Joke Limit Reached for today.", 400, nil)
	ErrInvalidUniqueCode             = NewAppError(1014, "Invalid unique code", 400, nil)
	ErrUniqueCodeAlreadyRedeemed     = NewAppError(1015, "Unique code already redeemed", 400, nil)
	ErrUniqueCodeExpired             = NewAppError(1016, "Unique code expired", 400, nil)
	ErrUniqueCodeNotActive           = NewAppError(1017, "Unique code not active", 400, nil)
	ErrFailedToRedeemUniqueCode      = NewAppError(1018, "Failed to redeem unique code", 400, nil)
	ErrDailyLimitExceeded            = NewAppError(1019, "Daily limit exceeded", 400, nil)
	ErrMaximumUGCContentLimitReached = NewAppError(1020, "Maximum UGC Limit Reached for today.", 400, nil)
)

// You can also create functions for specific error types if you need more customization
// For example:
func NewNotFoundError(message string, err error) *AppError {
	if message == "" {
		message = "The requested resource was not found."
	}
	return NewAppError(1001, message, 404, err)
}

func NewBadRequestError(message string, err error) *AppError {
	if message == "" {
		message = "The request is invalid."
	}
	return NewAppError(1002, message, 400, err)
}

func NewUnauthorizedError(message string, err error) *AppError {
	if message == "" {
		message = "Unauthorized."
	}
	return NewAppError(1004, message, 401, err)
}

func NewForbiddenError(message string, err error) *AppError {
	if message == "" {
		message = "Forbidden."
	}
	return NewAppError(1005, message, 403, err)
}

func NewConflictError(message string, err error) *AppError {
	if message == "" {
		message = "A conflict occurred."
	}
	return NewAppError(1006, message, 409, err)
}

func NewUnprocessableEntityError(message string, err error) *AppError {
	if message == "" {
		message = "The request could not be processed due to semantic errors."
	}
	return NewAppError(1007, message, 422, err)
}

func NewInternalServerError(message string, err error) *AppError {
	if message == "" {
		message = "An internal server error occurred."
	}
	return NewAppError(1003, message, 500, err)
}

func NewValidationErrors(message string, err error) *AppError {
	if message == "" {
		message = "Validation failed."
	}
	return NewAppError(1008, message, 400, err)
}

// NewValidationErrorWithFields creates a validation error with specific field errors.
func NewValidationErrorWithFields(message string, fieldErrors map[string][]string) *AppError {
	if message == "" {
		message = "Validation failed."
	}
	details := map[string]interface{}{
		"validation_errors": fieldErrors,
	}
	return NewAppErrorWithDetails(1008, message, 400, nil, details)
}

// NewValidationErrorWithSingleField creates a validation error for a single field.
func NewValidationErrorWithSingleField(field, message string) *AppError {
	fieldErrors := map[string][]string{
		field: {message},
	}
	return NewValidationErrorWithFields("", fieldErrors)
}

func NewTooManyRequestsError(message string, err error) *AppError {
	if message == "" {
		message = "Too many requests."
	}
	return NewAppError(1012, message, 429, err)
}
