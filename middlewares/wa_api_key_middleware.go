package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
)

func APIKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("x-api-key")
		cfg := config.GetConfig()

		logrus.Infof("API Key: %v\n", len(apiKey))
		if apiKey == "" {
			c.String(http.StatusUnauthorized, "Unauthorized Access")
			c.Abort()
			return
		}

		logrus.Infof("cfg.APIKey: %v\n", len(cfg.APIKey))
		logrus.Infof("Equality check: %v\n", cfg.APIKey == apiKey)
		if apiKey != cfg.APIKey {
			c.String(http.StatusUnauthorized, "Unauthorized Access")
			c.Abort()
			return
		}

		c.Next()
	}
}
