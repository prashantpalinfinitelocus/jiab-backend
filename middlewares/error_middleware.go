package middlewares

import (
	"net/http"

	"github.com/Infinite-Locus-Product/coca_cola_backend/errors"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check for errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err // Get the last error

			// Check if it's an AppError
			if appErr, ok := err.(*errors.AppError); ok {
				response := gin.H{
					"success": false,
					"code":    appErr.Code,
					"status":  appErr.Status,
					"message": appErr.Message,
				}

				// Add details if they exist
				if len(appErr.Details) > 0 {
					response["details"] = appErr.Details
				}

				c.JSON(appErr.Status, response)
				log.Errorf("ERROR: %v\n", err)
				return
			}

			// Handle other types of errors (e.g., built-in errors)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"code":    errors.ErrInternalServer.Code,
				"status":  errors.ErrInternalServer.Status,
				"message": errors.ErrInternalServer.Message,
			})
			log.Errorf("UNHANDLED ERROR: %v\n", err)
			return
		}
	}
}
