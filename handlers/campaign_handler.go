package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type CampaignHandler interface {
	SendProfileCompletionCampaign(c *gin.Context)
	SendContestAnnouncement(c *gin.Context)
}

type campaignHandler struct {
	campaignService services.CampaignService
}

func NewCampaignHandler(campaignService services.CampaignService) CampaignHandler {
	return &campaignHandler{campaignService: campaignService}
}

// SendProfileCompletionCampaign godoc
// @Summary Ops-triggered profile completion bulk campaign
// @Description Sends a coin-reward-attached "complete your profile" push to every eligible user (still incomplete, not sent within the weekly cooldown, re-checked at send time)
// @Tags Notifications
// @Accept json
// @Produce json
// @Param request body dtos.ProfileCompletionCampaignDTO true "Campaign content"
// @Success 200 {object} dtos.NotificationSendResult "Campaign run summary"
// @Security ApiKeyAuth
// @Router /notifications/profile-completion [post]
func (h *campaignHandler) SendProfileCompletionCampaign(c *gin.Context) {
	var req dtos.ProfileCompletionCampaignDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	// Detach from the request's cancellation signal: this campaign can take
	// far longer than an HTTP client is willing to hold a connection open for
	// (thousands of users, paginated in batches). Without this, the request
	// context gets canceled mid-run — by the client disconnecting, or any
	// proxy/load-balancer timeout in front of Cloud Run — and every DB call
	// for the rest of the run fails with "context canceled", which is exactly
	// what happened in production: eligible=2500 sent=0 failed=2500.
	result, err := h.campaignService.SendProfileCompletionCampaign(context.WithoutCancel(c.Request.Context()), req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// SendContestAnnouncement godoc
// @Summary Contest announcement broadcast
// @Description Broadcasts a contest launch or deadline-reminder announcement to all active users. Deduplicated per (contest_id, announcement_type) — the same contest can only get one launch and one reminder broadcast, ever.
// @Tags Notifications
// @Accept json
// @Produce json
// @Param request body dtos.ContestAnnouncementDTO true "Announcement content"
// @Success 200 {object} dtos.NotificationSendResult "Broadcast run summary"
// @Failure 409 {object} map[string]string "This contest/announcement_type has already been broadcast"
// @Security ApiKeyAuth
// @Router /notifications/broadcast-contest [post]
func (h *campaignHandler) SendContestAnnouncement(c *gin.Context) {
	var req dtos.ContestAnnouncementDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrors := utils.FormatValidationErrors(err)
		c.Error(errs.NewValidationErrorWithFields("", validationErrors))
		return
	}

	result, err := h.campaignService.SendContestAnnouncement(context.WithoutCancel(c.Request.Context()), req)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}
