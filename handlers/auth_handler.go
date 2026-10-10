package handlers

import (
	"net/http"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AuthHandler interface {
	RequestOTP(c *gin.Context)
	VerifyOTP(c *gin.Context)
	RefreshToken(c *gin.Context)
	SignUp(c *gin.Context)
	WAValidateContactNo(c *gin.Context)
	WARegister(c *gin.Context)
	WAToken(c *gin.Context)
	SSOInitiate(c *gin.Context)
	SSOAuthenticate(c *gin.Context)
}

type authHandler struct {
	authService services.AuthService
}

func NewAuthHandler(authService services.AuthService) AuthHandler {
	return &authHandler{authService: authService}
}

// RequestOTP godoc
// @Summary Request OTP
// @Description Request OTP for authentication
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dtos.RequestOTPRequest true "Request OTP"
// @Success 200 {object} utils.ResponseType[string] "OTP sent successfully"
// @Router /auth/request-otp [post]
func (h *authHandler) RequestOTP(c *gin.Context) {
	var req dtos.RequestOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	if err := h.authService.SendOTP(c, req.MobileNumber); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP sent successfully",
	})
}

// VerifyOTP godoc
// @Summary Verify OTP
// @Description Verify OTP for authentication
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dtos.VerifyOTPRequest true "Verify OTP"
// @Success 200 {object} utils.ResponseType[dtos.TokenResponse] "OTP verified successfully"
// @Router /auth/verify-otp [post]
func (h *authHandler) VerifyOTP(c *gin.Context) {
	var req dtos.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	tokenResponse, err := h.authService.VerifyOTP(c, req.MobileNumber, req.Otp)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tokenResponse)
}

// SignUp godoc
// @Summary Sign Up
// @Description Sign Up for authentication
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.SignUpRequest true "Sign Up"
// @Success 200 {object} utils.ResponseType[string] "User signed up successfully"
// @Router /auth/signup [post]
func (h *authHandler) SignUp(c *gin.Context) {
	var req dtos.SignUpRequest
	claims, err := utils.GetAndParseTempTokenFromHeader(c.Request)
	if err != nil {
		c.Error(err)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	tokenResponse, err := h.authService.SignUp(c, claims.MobileNumber, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tokenResponse)
}

// RefreshToken godoc
// @Summary Refresh Token
// @Description Refresh Token for authentication
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dtos.RefreshTokenRequest true "Refresh Token"
// @Success 200 {object} utils.ResponseType[dtos.TokenResponse] "Token refreshed successfully"
// @Router /auth/refresh-token [post]
func (h *authHandler) RefreshToken(c *gin.Context) {
	var req dtos.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	tokenResponse, err := h.authService.RefreshToken(c, req.RefreshToken)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tokenResponse)
}

// WAValidateContactNo godoc
// @Summary Validate Contact Number
// @Description Check if a contact number is already registered
// @Tags auth
// @Accept json
// @Produce json
// @Param contactNo query string true "Contact Number"
// @Success 200 {object} utils.ResponseType[int] "Valid contact number"
// @Success 409 {object} utils.ErrorType "Number already in use"
// @Router /auth/WA/validate-contact-no [post]
func (h *authHandler) WAValidateContactNo(c *gin.Context) {
	contactNo := c.Query("contactNo")
	if contactNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "Contact number is required",
			"succeeded":       false,
			"statusCode":      http.StatusBadRequest,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}

	_, err := h.authService.ValidateContactNo(c.Request.Context(), contactNo)
	if err != nil {
		// Handle all errors by extracting message and status from service layer
		if appErr, ok := err.(*errs.AppError); ok {
			c.JSON(appErr.Status, gin.H{
				"data":            0,
				"failed":          true,
				"message":         appErr.Message,
				"succeeded":       false,
				"statusCode":      appErr.Status,
				"page":            0,
				"totalPages":      0,
				"totalCount":      0,
				"hasPreviousPage": false,
				"hasNextPage":     false,
			})
			return
		}
		// Fallback for non-AppError errors
		c.JSON(http.StatusInternalServerError, gin.H{
			"data":            0,
			"failed":          true,
			"message":         "Internal server error",
			"succeeded":       false,
			"statusCode":      http.StatusInternalServerError,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":            1,
		"failed":          false,
		"message":         "Valid contact no.",
		"succeeded":       true,
		"statusCode":      http.StatusOK,
		"page":            0,
		"totalPages":      0,
		"totalCount":      0,
		"hasPreviousPage": false,
		"hasNextPage":     false,
	})
}

func (h *authHandler) WARegister(c *gin.Context) {
	var req dtos.WARegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errs.NewValidationErrorWithFields("Invalid request body", utils.FormatValidationErrors(err)))
		return
	}

	resp, err := h.authService.WARegister(c, req)
	if err != nil {
		// Handle all errors by extracting message and status from service layer
		if appErr, ok := err.(*errs.AppError); ok {
			c.JSON(appErr.Status, gin.H{
				"data":            nil,
				"failed":          true,
				"message":         appErr.Message,
				"succeeded":       false,
				"statusCode":      appErr.Status,
				"page":            0,
				"totalPages":      0,
				"totalCount":      0,
				"hasPreviousPage": false,
				"hasNextPage":     false,
			})
			return
		}
		// Fallback for non-AppError errors
		c.JSON(500, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "Internal server error",
			"succeeded":       false,
			"statusCode":      500,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}

	c.JSON(200, gin.H{
		"data":            resp,
		"failed":          false,
		"message":         "Authenticated",
		"succeeded":       true,
		"statusCode":      200,
		"page":            0,
		"totalPages":      0,
		"totalCount":      0,
		"hasPreviousPage": false,
		"hasNextPage":     false,
	})
}

func (h *authHandler) WAToken(c *gin.Context) {
	var req dtos.WATokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errs.NewValidationErrorWithFields("Invalid request body", utils.FormatValidationErrors(err)))
		return
	}

	resp, err := h.authService.WAToken(c, req.MobileNumber)
	if err != nil {
		// Handle all errors by extracting message and status from service layer
		if appErr, ok := err.(*errs.AppError); ok {
			c.JSON(appErr.Status, gin.H{
				"data":            nil,
				"failed":          true,
				"message":         appErr.Message,
				"succeeded":       false,
				"statusCode":      appErr.Status,
				"page":            0,
				"totalPages":      0,
				"totalCount":      0,
				"hasPreviousPage": false,
				"hasNextPage":     false,
			})
			return
		}
		// Fallback for non-AppError errors
		c.JSON(500, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "Internal server error",
			"succeeded":       false,
			"statusCode":      500,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}

	c.JSON(200, gin.H{
		"data":            resp,
		"failed":          false,
		"message":         "Authenticated",
		"succeeded":       true,
		"statusCode":      200,
		"page":            0,
		"totalPages":      0,
		"totalCount":      0,
		"hasPreviousPage": false,
		"hasNextPage":     false,
	})
}

// SSOInitiate godoc
// @Summary Initiate SSO flow (Partner endpoint)
// @Description Partner site initiates SSO by sending phone number and name; returns a short-lived SSO token and redirect URL
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dtos.SSOInitiateRequest true "User details from partner"
// @Success 200 {object} dtos.SSOInitiateResponse "SSO token generated successfully"
// @Router /auth/sso/initiate [post]
func (h *authHandler) SSOInitiate(c *gin.Context) {
	var req dtos.SSOInitiateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	resp, err := h.authService.SSOInitiate(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SSOAuthenticate godoc
// @Summary Authenticate via SSO token
// @Description Exchange a short-lived SSO token for JWT access and refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dtos.SSOAuthenticateRequest true "SSO token"
// @Success 200 {object} dtos.SSOAuthenticateResponse "Authentication successful"
// @Router /auth/sso/authenticate [post]
func (h *authHandler) SSOAuthenticate(c *gin.Context) {
	var req dtos.SSOAuthenticateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	resp, err := h.authService.SSOAuthenticate(c.Request.Context(), req.SSOToken)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
