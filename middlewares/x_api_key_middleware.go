package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
)

func XAPIKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("x-api-key")
		cfg := config.GetConfig()

		if apiKey == "" {
			c.String(http.StatusUnauthorized, "Unauthorized Access")
			c.Abort()
			return
		}

		if apiKey != cfg.XAPIKey {
			c.String(http.StatusUnauthorized, "Unauthorized Access")
			c.Abort()
			return
		}

		c.Next()
	}
}
