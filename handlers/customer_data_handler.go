package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
)

type CustomerDataHandler struct {
	cdsService services.CustomerDataService
}

func NewCustomerDataHandler(cdsService services.CustomerDataService) *CustomerDataHandler {
	return &CustomerDataHandler{
		cdsService: cdsService,
	}
}

// GetUserKOCID GET /api/user/kocid
func (h *CustomerDataHandler) GetUserKOCID(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	userEntity := user.(*entities.User)

	kocidResp, err := h.cdsService.GetUserKOCID(c.Request.Context(), userEntity.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve KOCID",
			"kocid": nil,
		})
		return
	}

	c.JSON(http.StatusOK, kocidResp)
}
