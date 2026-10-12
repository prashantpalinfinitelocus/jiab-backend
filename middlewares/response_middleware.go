package middlewares

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
)

type responseCapture struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r *responseCapture) Write(data []byte) (int, error) {
	r.body.Write(data)
	return len(data), nil
}

func SuccessResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip middleware for Swagger routes
		if isSwaggerRoute(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Create response capture
		capture := &responseCapture{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = capture

		// Process the request
		c.Next()

		// Handle the response
		statusCode := c.Writer.Status()
		contentType := capture.Header().Get("Content-Type")

		if statusCode >= 200 && statusCode < 300 && isJSONWrappable(contentType) {
			handleSuccessResponse(c, capture, statusCode)
		} else {
			// Binary/file responses (e.g. meme-generator's streamed image/gif)
			// must reach the client byte-for-byte - re-encoding them as a JSON
			// string here would corrupt them, since raw binary is rarely valid
			// UTF-8.
			c.Writer = capture.ResponseWriter
			if capture.body.Len() > 0 {
				_, _ = c.Writer.Write(capture.body.Bytes())
			}
		}
	}
}

// isJSONWrappable reports whether a response's Content-Type is safe to
// re-encode inside the {data, message, status, success} envelope. Empty
// means the handler never set one (typical for c.JSON, which sets it via
// gin's render step - by the time this runs, after c.Next(), it's already
// on the real ResponseWriter's header map), so that still gets wrapped.
func isJSONWrappable(contentType string) bool {
	if contentType == "" {
		return true
	}
	return strings.HasPrefix(contentType, "application/json")
}

func isSwaggerRoute(path string) bool {
	swaggerPaths := []string{
		"/swagger/index.html",
		"/swagger/doc.json",
		"/swagger/swagger-ui-bundle.js",
		"/swagger/swagger-ui-standalone-preset.js",
		"/swagger/swagger-ui.css",
		"/swagger/favicon-32x32.png",
		"/swagger/favicon-16x16.png",
	}

	for _, swaggerPath := range swaggerPaths {
		if path == swaggerPath {
			return true
		}
	}

	return len(path) > 8 && path[:8] == "/swagger"
}

func handleSuccessResponse(c *gin.Context, capture *responseCapture, statusCode int) {
	// nosemgrep: go.lang.security.deserialization.unsafe-deserialization-interface.go-unsafe-deserialization-interface -- generic response envelope: handlers return arbitrary JSON bodies that are re-wrapped unchanged (own output, not external input)
	var data interface{}
	message := "Request successful"

	// Parse captured response
	if capture.body.Len() > 0 {
		bodyBytes := capture.body.Bytes()
		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			data = string(bodyBytes)
		} else {
			// Extract message if it exists
			if dataMap, ok := data.(map[string]interface{}); ok {
				if msg, exists := dataMap["message"]; exists {
					if msgStr, ok := msg.(string); ok {
						message = msgStr
						delete(dataMap, "message")
						if len(dataMap) == 0 {
							data = nil
						} else {
							data = dataMap
						}
					}
				}
			}
		}
	}

	// Create standardized response
	response := gin.H{
		"success": true,
		"status":  statusCode,
		"message": message,
	}

	if data != nil {
		response["data"] = data
	}

	// Write the final response
	c.Writer = capture.ResponseWriter
	c.Header("Content-Type", "application/json")
	c.JSON(statusCode, response)
}
