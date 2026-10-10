package utils

import "github.com/gin-gonic/gin"

type ResponseType[T any] struct {
	Success bool   `json:"success"`
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

type ErrorType struct {
	Success bool                `json:"success"`
	Code    string              `json:"code"`
	Status  int                 `json:"status"`
	Message string              `json:"message"`
	Details map[string][]string `json:"details,omitempty"`
}

type ReponseTypeAny struct {
	Success bool                   `json:"success"`
	Status  int                    `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// SendWAResponse is a helper to standardize responses for WA routes.
// It sends a JSON response with a structured format containing data and a message.
func SendWAResponse(c *gin.Context, code int, data interface{}, message string) {
	c.JSON(code, gin.H{
		"data":    data,
		"message": message,
	})
}
