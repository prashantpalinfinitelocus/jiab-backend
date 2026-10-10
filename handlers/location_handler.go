package handlers

import (
	"net/http"
	"strconv"

	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/gin-gonic/gin"
)

type LocationHandler struct {
	locationService services.LocationService
}

func NewLocationHandler(locationService services.LocationService) *LocationHandler {
	return &LocationHandler{
		locationService: locationService,
	}
}

// GetLocationByPincode godoc
// @Summary Get Location Details by Pincode
// @Description Retrieves state and city for a given pincode
// @Tags location
// @Accept json
// @Produce json
// @Param pincode query int true "Pincode"
// @Success 200 {object} dtos.LocationResponseDTO
// @Failure 400 {object} utils.ErrorType "Invalid pincode format"
// @Failure 404 {object} utils.ErrorType "Pincode not found"
// @Failure 500 {object} utils.ErrorType
// @Router /api/v1/location/pincode [get]
func (h *LocationHandler) GetLocationByPincode(c *gin.Context) {
	pincodeStr := c.Query("pincode")
	if pincodeStr == "" {
		c.Error(errs.NewBadRequestError("pincode query parameter is required", nil))
		return
	}

	pincode, err := strconv.Atoi(pincodeStr)
	if err != nil {
		c.Error(errs.NewBadRequestError("invalid pincode format, must be a number", err))
		return
	}

	location, err := h.locationService.GetLocationByPincode(c, pincode)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, location)
}
