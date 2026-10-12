package middlewares

import (
	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
)

func WASuccessResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		capture := &responseCapture{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = capture

		c.Next()

		statusCode := c.Writer.Status()

		if statusCode >= 200 && statusCode < 300 {
			handleWASuccessResponse(c, capture, statusCode)
		} else {
			c.Writer = capture.ResponseWriter
			if capture.body.Len() > 0 {
				_, _ = c.Writer.Write(capture.body.Bytes())
			}
		}
	}
}

func handleWASuccessResponse(c *gin.Context, capture *responseCapture, statusCode int) {
	var data interface{}
	message := ""
	failed := false
	succeeded := true
	waStatusCode := statusCode
	page := 0
	totalPages := 0
	totalCount := 0
	hasPreviousPage := false
	hasNextPage := false

	if capture.body.Len() > 0 {
		bodyBytes := capture.body.Bytes()
		var dataMap map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &dataMap); err == nil {
			if v, ok := dataMap["data"]; ok {
				data = v
			}
			if v, ok := dataMap["message"]; ok {
				if msgStr, ok := v.(string); ok {
					message = msgStr
				}
			}
			if v, ok := dataMap["failed"]; ok {
				if failBool, ok := v.(bool); ok {
					failed = failBool
				}
			}
			if v, ok := dataMap["succeeded"]; ok {
				if succBool, ok := v.(bool); ok {
					succeeded = succBool
				}
			}
			if v, ok := dataMap["statusCode"]; ok {
				if code, ok := v.(float64); ok {
					waStatusCode = int(code)
				}
			}
			if v, ok := dataMap["page"]; ok {
				if p, ok := v.(float64); ok {
					page = int(p)
				}
			}
			if v, ok := dataMap["totalPages"]; ok {
				if tp, ok := v.(float64); ok {
					totalPages = int(tp)
				}
			}
			if v, ok := dataMap["totalCount"]; ok {
				if tc, ok := v.(float64); ok {
					totalCount = int(tc)
				}
			}
			if v, ok := dataMap["hasPreviousPage"]; ok {
				if hpp, ok := v.(bool); ok {
					hasPreviousPage = hpp
				}
			}
			if v, ok := dataMap["hasNextPage"]; ok {
				if hnp, ok := v.(bool); ok {
					hasNextPage = hnp
				}
			}
		} else {
			// nosemgrep: go.lang.security.deserialization.unsafe-deserialization-interface.go-unsafe-deserialization-interface -- generic response envelope: re-wraps this service's own handler output unchanged
			var rawData interface{}
			if err := json.Unmarshal(bodyBytes, &rawData); err == nil {
				data = rawData
			} else {
				data = string(bodyBytes)
			}
		}
	}

	waResp := gin.H{
		"data":            data,
		"failed":          failed,
		"message":         message,
		"succeeded":       succeeded,
		"statusCode":      waStatusCode,
		"page":            page,
		"totalPages":      totalPages,
		"totalCount":      totalCount,
		"hasPreviousPage": hasPreviousPage,
		"hasNextPage":     hasNextPage,
	}

	c.Writer = capture.ResponseWriter
	c.Header("Content-Type", "application/json")
	c.JSON(waStatusCode, waResp)
}
