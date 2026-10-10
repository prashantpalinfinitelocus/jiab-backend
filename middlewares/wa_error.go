package middlewares

import (
	"net/http"

	"github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func WAErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			if appErr, ok := err.(*errors.AppError); ok {
				if appErr.Status == http.StatusUnauthorized {
					c.String(http.StatusUnauthorized, "Unauthorized Access")
					c.Abort()
					return
				}

				waResp := gin.H{
					"data":       0,
					"failed":     true,
					"message":    appErr.Message,
					"succeeded":  false,
					"statusCode": appErr.Status,
				}
				if len(appErr.Details) > 0 {
					waResp["details"] = appErr.Details
				}
				c.JSON(appErr.Status, waResp)
				log.Errorf("WA ERROR: %v\n", err)
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"data":       0,
				"failed":     true,
				"message":    errors.ErrInternalServer.Message,
				"succeeded":  false,
				"statusCode": errors.ErrInternalServer.Status,
			})
			log.Errorf("WA UNHANDLED ERROR: %v\n", err)
			return
		}
	}
}
