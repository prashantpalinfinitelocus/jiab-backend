package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
)

type AdminMemeModerationHandler struct {
	service services.MemeService
}

func NewAdminMemeModerationHandler(service services.MemeService) *AdminMemeModerationHandler {
	return &AdminMemeModerationHandler{service: service}
}

// @Summary List pending memes for moderation
// @Description Get a list of pending memes awaiting admin moderation
// @Tags Admin Meme Moderation
// @Accept json
// @Produce json
// @Param limit query int false "Number of results to return" default(15)
// @Param cursor query string false "Cursor for pagination"
// @Success 200 {object} dtos.AdminPendingMemesResponse "List of pending memes"
// @Router /admin/memes/pending [get]
func (h *AdminMemeModerationHandler) ListPending(c *gin.Context) {
	limit := 15
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	var cursor *string
	if cur := c.Query("cursor"); cur != "" {
		cursor = &cur
	}

	resp, err := h.service.AdminListPending(c, limit, cursor)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Approve a meme
// @Description Approve a pending meme for publication
// @Tags Admin Meme Moderation
// @Accept json
// @Produce json
// @Param id path string true "Meme ID"
// @Param request body dtos.ApproveMemeRequest true "Approval details"
// @Success 200 {object} dtos.ApproveMemeResponse "Meme approved successfully"
// @Router /admin/memes/{id}/approve [post]
func (h *AdminMemeModerationHandler) Approve(c *gin.Context) {
	id := c.Param("id")
	var req dtos.ApproveMemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	resp, err := h.service.AdminApprove(c, id, req.LanguageID, req.Notes)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Reject a meme
// @Description Reject a pending meme with a reason
// @Tags Admin Meme Moderation
// @Accept json
// @Produce json
// @Param id path string true "Meme ID"
// @Param request body dtos.RejectMemeRequest true "Rejection details"
// @Success 200 {object} dtos.RejectMemeResponse "Meme rejected successfully"
// @Router /admin/memes/{id}/reject [post]
func (h *AdminMemeModerationHandler) Reject(c *gin.Context) {
	id := c.Param("id")
	var req dtos.RejectMemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	resp, err := h.service.AdminReject(c, id, req.Reason)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
