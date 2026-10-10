package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Custom errors
var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrInvalidTokenType  = errors.New("invalid token type")
	ErrMissingToken      = errors.New("missing authorization token")
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")
)

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken        string    `json:"access_token"`
	RefreshToken       string    `json:"refresh_token"`
	AccessTokenExpiry  int64     `json:"access_token_expiry"`
	RefreshTokenExpiry time.Time `json:"refresh_token_expiry"`
	TokenType          string    `json:"token_type"`
}

// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	UserID       string `json:"user_id"`
	MobileNumber string `json:"mobile_number"`
	TokenType    string `json:"token_type"`
	jwt.RegisteredClaims
}

// TempTokenClaims represents the claims in temporary JWT tokens
type TempTokenClaims struct {
	UserID       string `json:"user_id,omitempty"`
	MobileNumber string `json:"mobile_number,omitempty"`
	Purpose      string `json:"purpose"`
	TokenType    string `json:"token_type"`
	jwt.RegisteredClaims
}

// TempTokenOptions represents options for creating temporary tokens
type TempTokenOptions struct {
	MobileNumber string
}

// GenerateTokenPair generates both access and refresh tokens for a user
func GenerateTokenPair(userID, mobileNumber string) (*TokenPair, error) {
	cfg := config.GetConfig()

	// Generate Access Token
	accessToken, err := generateToken(userID, mobileNumber, "access", cfg.JwtConfig.AccessTokenExpiry)
	if err != nil {
		return nil, err
	}

	// Generate Refresh Token
	refreshToken, err := generateMinimalToken(userID, mobileNumber, "refresh", cfg.JwtConfig.RefreshTokenExpiry)
	if err != nil {
		return nil, err
	}

	// Calculate expiry times as time.Time objects
	refreshTokenExpiryTime := time.Now().Add(time.Duration(cfg.JwtConfig.RefreshTokenExpiry) * time.Minute)

	return &TokenPair{
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		AccessTokenExpiry:  int64(cfg.JwtConfig.AccessTokenExpiry * 60),
		RefreshTokenExpiry: refreshTokenExpiryTime,
		TokenType:          "Bearer",
	}, nil
}

// GenerateAccessToken generates only an access token
func GenerateAccessToken(userID, mobileNumber string) (string, error) {
	cfg := config.GetConfig()
	return generateToken(userID, mobileNumber, "access", cfg.JwtConfig.AccessTokenExpiry)
}

// GenerateRefreshToken generates only a refresh token
func GenerateRefreshToken(userID, mobileNumber string) (string, error) {
	cfg := config.GetConfig()
	return generateMinimalToken(userID, mobileNumber, "refresh", cfg.JwtConfig.RefreshTokenExpiry)
}

// generateToken is a helper function to generate JWT tokens
func generateToken(userID, mobileNumber, tokenType string, expiryMinutes int) (string, error) {
	cfg := config.GetConfig()

	// Create claims
	claims := JWTClaims{
		UserID:       userID,
		MobileNumber: mobileNumber,
		TokenType:    tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID,
			Issuer:    "coca_cola_backend",
			Audience:  []string{"coca_cola_app"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryMinutes) * time.Minute)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(cfg.JwtConfig.SecretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// generateMinimalToken generates an opaque, unguessable refresh-token string.
// Its only requirement is unguessability: unlike the access token, nothing
// ever decodes or recomputes this value to re-derive userID/expiry from it -
// RefreshToken (services/auth_service.go) looks the literal string up in the
// user_refresh_tokens table and enforces expiry from that row's own column.
// It's an HMAC (keyed by the server secret) rather than a plain hash so the
// output isn't just a public hash of guessable inputs (userID + a timestamp).
func generateMinimalToken(userID, mobileNumber, tokenType string, expiryMinutes int) (string, error) {
	cfg := config.GetConfig()
	expiryTime := time.Now().Add(time.Duration(expiryMinutes) * time.Minute).Unix()
	data := fmt.Sprintf("%s_%d_%s_%s", userID, expiryTime, mobileNumber, tokenType)

	mac := hmac.New(sha256.New, []byte(cfg.JwtConfig.SecretKey))
	mac.Write([]byte(data))
	tokenString := strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))

	return tokenString, nil
}

// RefreshAccessToken generates a new access token using a valid refresh token
func RefreshAccessToken(refreshTokenString string, userID string, mobileNumber string) (*TokenPair, error) {
	// Generate new token pair
	return GenerateTokenPair(userID, mobileNumber)
}

// GenerateTempToken generates a temporary JWT token for specific purposes
func GenerateTempToken(options TempTokenOptions) (string, error) {
	cfg := config.GetConfig()

	// Create claims
	claims := TempTokenClaims{
		MobileNumber: options.MobileNumber,
		TokenType:    "temp",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   options.MobileNumber,
			Issuer:    "coca_cola_backend",
			Audience:  []string{"coca_cola_app"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JwtConfig.TempTokenExpiry) * time.Minute)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(cfg.JwtConfig.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateTempToken validates and parses a temporary JWT token
func ValidateTempToken(tokenString string) (*TempTokenClaims, error) {
	cfg := config.GetConfig()

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &TempTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(cfg.JwtConfig.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	// Extract claims
	if claims, ok := token.Claims.(*TempTokenClaims); ok && token.Valid {
		// Verify it's a temp token
		if claims.TokenType != "temp" {
			return nil, ErrInvalidTokenType
		}
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// GenerateTempAccessToken creates a temporary access token
func GenerateTempAccessToken(mobileNumber string) (string, error) {
	return GenerateTempToken(TempTokenOptions{
		MobileNumber: mobileNumber,
	})
}

// GenerateSecureToken generates a cryptographically secure random alphanumeric token of the given length
func GenerateSecureToken(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	token := make([]byte, length)
	for i := range token {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate secure token: %w", err)
		}
		token[i] = charset[n.Int64()]
	}
	return string(token), nil
}

// GetAndParseTempTokenFromHeader extracts and validates a temporary token from the Authorization header
func GetAndParseTempTokenFromHeader(r *http.Request) (*TempTokenClaims, error) {
	// A missing/malformed header or an invalid/expired temp token are both
	// the caller's fault, not ours - map both to 401 rather than 500. A
	// StackHawk DAST scan's unauthenticated requests (no Authorization
	// header at all) were hitting the 500 branch here, flagged as
	// "Application Error Disclosure".
	tokenString, err := extractTokenFromHeader(r)
	if err != nil {
		return nil, errs.NewUnauthorizedError("Failed to extract token from header", err)
	}

	// Validate and parse the temp token
	claims, err := ValidateTempToken(tokenString)
	if err != nil {
		return nil, errs.NewUnauthorizedError("Failed to validate temp token", err)
	}

	return claims, nil
}

// extractTokenFromHeader extracts the JWT token from the Authorization header
func extractTokenFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrMissingToken
	}

	// Check if header starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", ErrInvalidAuthHeader
	}

	// Extract token (remove "Bearer " prefix)
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return "", ErrMissingToken
	}

	return tokenString, nil
}
