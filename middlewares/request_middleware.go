package middlewares

import (
	"context"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
	"github.com/sirupsen/logrus"
)

func GetRequestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value("requestID").(string)
	if !ok {
		return ""
	}
	return requestID
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := utils.SanitizeForLog(c.Request.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = ulid.Make().String()
		}
		c.Set("requestID", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		user, _ := c.Get("user")
		path := utils.SanitizeForLog(c.Request.URL.Path)

		// Add the hook to the logrus logger
		logrus.AddHook(&LogRequestIDHook{Context: c})

		// Log the request
		if user == nil {
			logrus.WithFields(logrus.Fields{
				"requestID": requestID,
				"method":    c.Request.Method,
				"path":      path,
			}).Info("Request started")

		} else {
			logrus.WithFields(logrus.Fields{
				"requestID": requestID,
				"method":    c.Request.Method,
				"path":      path,
				"userID":    user.(*entities.User).ID,
			}).Info("Request started")
		}

		logrus.WithFields(logrus.Fields{
			"requestID": requestID,
			"method":    c.Request.Method,
			"path":      path,
		}).Info("Request started")

		c.Next()

		// Log the response
		logrus.WithFields(logrus.Fields{
			"requestID": requestID,
			"status":    c.Writer.Status(),
		}).Info("Request completed")
	}
}
