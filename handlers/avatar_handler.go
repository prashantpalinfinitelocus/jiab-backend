package handlers

import (
	"net/http"

	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	_ "github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/gin-gonic/gin"
)

type AvatarHandler struct {
	avatarService services.AvatarService
}

func NewAvatarHandler(avatarService services.AvatarService) *AvatarHandler {
	return &AvatarHandler{
		avatarService: avatarService,
	}
}

// GetAvatars godoc
// @Summary Get all active avatars
// @Description Retrieves all active avatars from the database
// @Tags avatars
// @Accept json
// @Produce json
// @Success 200 {object} utils.ResponseType[[]entities.Avatar]
// @Failure 500 {object} utils.ErrorType
// @Router /api/v1/avatars [get]
func (h *AvatarHandler) GetAvatars(c *gin.Context) {
	avatars, err := h.avatarService.GetAllAvatars(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   avatars,
	})
}
