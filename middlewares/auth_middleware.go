package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

func ServiceInjector(userService services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userService", userService)
		c.Next()
	}
}

type JwtClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userService, exists := c.Get("userService")
		if !exists {
			logrus.Error("userService not found in context")
			c.Error(errors.NewInternalServerError("userService not found in context", nil))
			c.Abort()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			authenticateUser(c, userService.(services.UserService))
			if c.IsAborted() {
				return
			}
		}

		c.Next()
	}
}

func GeneralAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userService, exists := c.Get("userService")
		if !exists {
			logrus.Error("userService not found in context")
			c.Error(errors.NewInternalServerError("userService not found in context", nil))
			c.Abort()
			return
		}
		authenticateUser(c, userService.(services.UserService))
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodDelete || c.Request.Method == http.MethodPatch {
			user, _ := c.Get("user")
			if user != nil && !user.(*entities.User).IsActive {
				c.Error(errors.NewUnauthorizedError("Your Account is Inactive", nil))
				c.Abort()
				return
			}
		}

	}
}

// WAGeneralAuth is a custom authentication middleware for WA routes
// that returns proper WA error response format instead of calling c.Abort()
func WAGeneralAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userService, exists := c.Get("userService")
		if !exists {
			logrus.Error("userService not found in context")
			c.JSON(500, gin.H{
				"Data":            nil,
				"Failed":          true,
				"Message":         "Internal server error",
				"Succeeded":       false,
				"StatusCode":      500,
				"Page":            0,
				"TotalPages":      0,
				"TotalCount":      0,
				"HasPreviousPage": false,
				"HasNextPage":     false,
			})
			c.Abort()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logrus.Error("Authorization header is missing")
			c.JSON(401, gin.H{
				"Data":            nil,
				"Failed":          true,
				"Message":         "Authorization header is missing",
				"Succeeded":       false,
				"StatusCode":      401,
				"Page":            0,
				"TotalPages":      0,
				"TotalCount":      0,
				"HasPreviousPage": false,
				"HasNextPage":     false,
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &utils.JWTClaims{}

		// Parse the JWT token
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.GetConfig().JwtConfig.SecretKey), nil
		})

		if err != nil || !token.Valid {
			logrus.Error(err)
			c.JSON(401, gin.H{
				"Data":            nil,
				"Failed":          true,
				"Message":         "You are not Authorized",
				"Succeeded":       false,
				"StatusCode":      401,
				"Page":            0,
				"TotalPages":      0,
				"TotalCount":      0,
				"HasPreviousPage": false,
				"HasNextPage":     false,
			})
			c.Abort()
			return
		}

		// Fetch the user based on the claims.UserID
		userID := claims.UserID
		user, derr := userService.(services.UserService).GetUser(c, userID)
		if derr != nil {
			c.JSON(500, gin.H{
				"Data":            nil,
				"Failed":          true,
				"Message":         "Internal server error",
				"Succeeded":       false,
				"StatusCode":      500,
				"Page":            0,
				"TotalPages":      0,
				"TotalCount":      0,
				"HasPreviousPage": false,
				"HasNextPage":     false,
			})
			c.Abort()
			return
		}

		// Check if user is active for POST/PUT/DELETE/PATCH requests
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodDelete || c.Request.Method == http.MethodPatch {
			if user != nil && !user.IsActive {
				c.JSON(401, gin.H{
					"Data":            nil,
					"Failed":          true,
					"Message":         "Your Account is Inactive",
					"Succeeded":       false,
					"StatusCode":      401,
					"Page":            0,
					"TotalPages":      0,
					"TotalCount":      0,
					"HasPreviousPage": false,
					"HasNextPage":     false,
				})
				c.Abort()
				return
			}
		}

		// Set user in context and proceed
		c.Set("user", user)
		c.Next()
	}
}

func authenticateUser(c *gin.Context, userService services.UserService) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		logrus.Error("Authorization header is missing")
		c.Error(errors.NewUnauthorizedError("Authorization header is missing", nil))
		c.Abort()
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims := &utils.JWTClaims{}

	// Parse the JWT token
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.GetConfig().JwtConfig.SecretKey), nil
	})

	if err != nil || !token.Valid {
		logrus.Error(err)
		c.Error(errors.NewUnauthorizedError("Unauthorized", nil))
		c.Abort()
		return
	}

	// Fetch the user based on the claims.UserID
	userID := claims.UserID
	if userID == "" {
		c.Error(errors.NewUnauthorizedError("Unauthorized", nil))
		c.Abort()
		return
	}
	user, derr := userService.GetUser(c, userID)
	if derr != nil {
		c.Error(derr)
		c.Abort()
		return
	}
	// Set user in context and proceed
	c.Set("user", user)
	return
}
