package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	_ "github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/gin-gonic/gin"
)

type VoucherHandler struct {
	voucherService services.VoucherService
}

func NewVoucherHandler(voucherService services.VoucherService) *VoucherHandler {
	return &VoucherHandler{voucherService: voucherService}
}

// @Summary Get voucher info
// @Description Get voucher info
// @Tags Voucher
// @Accept json
// @Produce json
// @Param language_id query string false "Language ID"
// @Security BearerAuth
// @Success 200 {object} utils.ResponseType[[]dtos.VoucherInfoDTO] "Voucher info"
// @Router /voucher/info [get]
func (h *VoucherHandler) GetVoucherInfo(c *gin.Context) {
	user, _ := c.Get("user")
	userId := user.(*entities.User).ID
	languageIDStr, exists := c.GetQuery("language_id")

	var languageID int
	if exists {
		var err error
		languageID, err = strconv.Atoi(languageIDStr)
		if err != nil {
			c.Error(errs.NewBadRequestError("Invalid language_id", fmt.Errorf("language_id must be an integer")))
			return
		}
	} else {
		languageID = constants.DEFAULT_LANGUAGE_ID
	}

	voucherInfos, err := h.voucherService.GetVoucherInfo(c, userId, languageID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, voucherInfos)
}
