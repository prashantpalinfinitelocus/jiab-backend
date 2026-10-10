package handlers

import (
	"net/http"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ReferralHandler struct {
	referralService services.ReferralService
}

func NewReferralHandler(referralService services.ReferralService) *ReferralHandler {
	return &ReferralHandler{
		referralService: referralService,
	}
}

// GenerateReferral godoc
// @Summary Generate a new referral code
// @Description Generate a new referral code for a user to share
// @Tags referral
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.GenerateReferralRequest true "Generate referral request"
// @Success 200 {object} utils.ResponseType[dtos.ReferralResponse] "Referral generated successfully"
// @Failure 400 {object} utils.ErrorType "Bad request"
// @Router /referral [post]
func (h *ReferralHandler) GenerateReferral(c *gin.Context) {
	logrus.Info("Generating referral code")
	var req dtos.GenerateReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("invalid request body: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	user, _ := c.Get("user")
	userID := user.(*entities.User).ID
	mobileNumber := user.(*entities.User).PhoneNumber

	referral, err := h.referralService.GenerateReferral(c.Request.Context(), userID, mobileNumber, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, referral)
}

// GetUserReferrals godoc
// @Summary Get all user referrals
// @Description Get all referrals created by the user
// @Tags referral
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param status query string false "Status"
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.ResponseType[dtos.ReferralListResponse] "Referrals retrieved successfully"
// @Failure 400 {object} utils.ErrorType "Internal server error"
// @Router /referral/invitees [get]
func (h *ReferralHandler) GetUserReferrals(c *gin.Context) {
	user, _ := c.Get("user")
	userID := user.(*entities.User).ID
	var filter dtos.ReferralFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		logrus.Errorf("invalid filters: %v", err)
		c.Error(errs.NewBadRequestError("Invalid filters", err))
		return
	}

	referrals, err := h.referralService.GetUserReferrals(c.Request.Context(), userID, filter)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, referrals)
}

// SendReferralAgain godoc
// @Summary Send referral code again
// @Description Resend an existing referral code
// @Tags referral
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.SendReferralAgainRequest true "Send referral again request"
// @Success 200 {object} utils.ResponseType[string] "Referral code sent successfully"
// @Failure 400 {object} utils.ErrorType "Bad request"
// @Router /referral/send-again [post]
func (h *ReferralHandler) SendReferralAgain(c *gin.Context) {
	user, _ := c.Get("user")
	userID := user.(*entities.User).ID
	var req dtos.SendReferralAgainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("invalid request body: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	referralCode, err := h.referralService.SendReferralAgain(c.Request.Context(), userID, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Referral code sent successfully",
		"referralCode": *referralCode,
	})
}

// VerifyReferral godoc
// @Summary Verify referral code
// @Description Verify and accept a referral code
// @Tags referral
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.VerifyReferralRequest true "Verify referral request"
// @Success 200 {object} utils.ResponseType[string] "Referral code verified successfully"
// @Router /referral/verify [post]
func (h *ReferralHandler) VerifyReferral(c *gin.Context) {
	user, _ := c.Get("user")
	userData := user.(*entities.User)
	var req dtos.VerifyReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("invalid request body: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	referrerPhoneNumber, err := h.referralService.VerifyReferral(c.Request.Context(), userData, req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Referral code verified successfully",
		"phone_number": referrerPhoneNumber,
	})
}
