package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// StripMainPrefix is a Gin middleware that removes the /main prefix from the request URL path.
func StripMainPrefix() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/main") {
			c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/main")
			if c.Request.URL.Path == "" {
				c.Request.URL.Path = "/"
			}
		}
		c.Next()
	}
} 