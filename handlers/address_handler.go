package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type AddressService interface {
	GetAddresses(ctx *gin.Context)
	AddAddress(ctx *gin.Context)
	EditAddress(ctx *gin.Context)
	DeleteAddress(ctx *gin.Context)
}

type AddressHandler struct {
	userService services.UserService
}

func NewAddressHandler(userService services.UserService) *AddressHandler {
	return &AddressHandler{
		userService: userService,
	}
}

// GetAddresses godoc
// @Summary Get Addresses
// @Description Get all addresses for the user
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.ResponseType[[]dtos.AddressResponseDTO] "Addresses retrieved successfully"
// @Router /profile/address [get]
func (h *AddressHandler) GetAddresses(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)
	addresses, err := h.userService.GetUserAddresses(ctx, userEntity.ID)
	if err != nil {
		ctx.Error(errs.NewInternalServerError("Failed to fetch addresses", fmt.Errorf("error fetching addresses for user %s: %w", userEntity.ID, err)))
		return
	}

	ctx.JSON(http.StatusOK, addresses)
}

// AddAddress godoc
// @Summary Add Address
// @Description Add a new address for the user
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.AddressRequestDTO true "Add Address Request"
// @Success 201 {object} utils.ResponseType[dtos.AddressResponseDTO] "Address added successfully"
// @Router /profile/address [post]
func (h *AddressHandler) AddAddress(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)
	var req dtos.AddressRequestDTO

	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		ctx.Error(errs.NewValidationErrorWithFields("Invalid request body", validationErrors))
		return
	}

	// Validate required fields with detailed error messages
	var missingFields []string
	if req.Address1 == "" {
		missingFields = append(missingFields, "address1")
	}
	if req.Pincode == 0 {
		missingFields = append(missingFields, "pincode")
	}
	if req.State == "" {
		missingFields = append(missingFields, "state")
	}
	if req.City == "" {
		missingFields = append(missingFields, "city")
	}
	if len(missingFields) > 0 {
		ctx.Error(errs.NewBadRequestError(fmt.Sprintf("Missing required fields: %s", strings.Join(missingFields, ", ")),
			fmt.Errorf("missing required fields for user %s: %s", userEntity.ID, strings.Join(missingFields, ", "))))
		return
	}

	// Validate pincode format
	if req.Pincode < 100000 || req.Pincode > 999999 {
		ctx.Error(errs.NewBadRequestError("Invalid pincode format. Must be a 6-digit number",
			fmt.Errorf("invalid pincode format provided: %d", req.Pincode)))
		return
	}

	address, err := h.userService.AddUserAddress(ctx, userEntity.ID, req)
	if err != nil {
		// Handle specific error cases
		switch {
		case strings.Contains(err.Error(), "state"):
			ctx.Error(errs.NewBadRequestError(err.Error(),
				fmt.Errorf("state validation error for user %s: %w", userEntity.ID, err)))
		case strings.Contains(err.Error(), "city"):
			ctx.Error(errs.NewBadRequestError(err.Error(),
				fmt.Errorf("city validation error for user %s: %w", userEntity.ID, err)))
		case strings.Contains(err.Error(), "pincode"):
			ctx.Error(errs.NewBadRequestError(err.Error(),
				fmt.Errorf("pincode validation error for user %s: %w", userEntity.ID, err)))
		case strings.Contains(err.Error(), "not deliverable"):
			ctx.Error(errs.NewBadRequestError(err.Error(),
				fmt.Errorf("pincode deliverability error for user %s: %w", userEntity.ID, err)))
		case err.Error() == "user not found":
			ctx.Error(errs.NewNotFoundError("User not found",
				fmt.Errorf("no user found with ID: %s", userEntity.ID)))
		default:
			ctx.Error(errs.NewInternalServerError("Failed to add address",
				fmt.Errorf("error adding address for user %s: %w", userEntity.ID, err)))
		}
		return
	}

	if address == nil {
		ctx.Error(errs.NewNotFoundError("Failed to create address",
			fmt.Errorf("address creation failed for user %s", userEntity.ID)))
		return
	}

	ctx.JSON(http.StatusCreated, address)
}

// EditAddress godoc
// @Summary Edit Address
// @Description Edit an existing address for the user
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param addressId path string true "Address ID"
// @Param request body dtos.AddressRequestDTO true "Edit Address Request"
// @Success 200 {object} utils.ResponseType[dtos.AddressResponseDTO] "Address updated successfully"
// @Router /profile/address/{addressId} [put]
func (h *AddressHandler) EditAddress(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)

	addressID := ctx.Param("addressId")
	if addressID == "" {
		ctx.Error(errs.NewBadRequestError("Address ID is required",
			fmt.Errorf("missing address ID in request for user %s", userEntity.ID)))
		return
	}

	var req dtos.AddressRequestDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		ctx.Error(errs.NewValidationErrorWithFields("Invalid request body", validationErrors))
		return
	}

	// Validate required fields
	if req.Address1 == "" || req.Pincode == 0 || req.State == "" || req.City == "" {
		ctx.Error(errs.NewBadRequestError("Missing required fields: address1, pincode, state, and city are required",
			fmt.Errorf("missing required fields for address %s of user %s", addressID, userEntity.ID)))
		return
	}

	address, err := h.userService.EditUserAddress(ctx, userEntity.ID, addressID, req)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "state"):
			ctx.Error(errs.NewBadRequestError(err.Error(),
				fmt.Errorf("state validation error for address %s of user %s: %w", addressID, userEntity.ID, err)))
		case strings.Contains(err.Error(), "city"):
			ctx.Error(errs.NewBadRequestError(err.Error(),
				fmt.Errorf("city validation error for address %s of user %s: %w", addressID, userEntity.ID, err)))
		case strings.Contains(err.Error(), "pincode"):
			ctx.Error(errs.NewBadRequestError(err.Error(),
				fmt.Errorf("pincode validation error for address %s of user %s: %w", addressID, userEntity.ID, err)))
		case strings.Contains(err.Error(), "not deliverable"):
			ctx.Error(errs.NewBadRequestError(err.Error(),
				fmt.Errorf("pincode deliverability error for address %s of user %s: %w", addressID, userEntity.ID, err)))
		case err.Error() == "user not found":
			ctx.Error(errs.NewNotFoundError("User not found",
				fmt.Errorf("no user found with ID: %s", userEntity.ID)))
		case err.Error() == "address not found":
			ctx.Error(errs.NewNotFoundError("Address not found",
				fmt.Errorf("no address found with ID: %s for user %s", addressID, userEntity.ID)))
		default:
			ctx.Error(errs.NewInternalServerError("Failed to update address",
				fmt.Errorf("error updating address %s for user %s: %w", addressID, userEntity.ID, err)))
		}
		return
	}

	if address == nil {
		ctx.Error(errs.NewNotFoundError("User not found",
			fmt.Errorf("no user found with ID: %s", userEntity.ID)))
		return
	}

	ctx.JSON(http.StatusOK, address)
}

// DeleteAddress godoc
// @Summary Delete Address
// @Description Delete an existing address for the user
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param addressId path string true "Address ID"
// @Success 200 {object} utils.ResponseType[string] "Address deleted successfully"
// @Router /profile/address/{addressId} [delete]
func (h *AddressHandler) DeleteAddress(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)

	addressID := ctx.Param("addressId")
	if addressID == "" {
		ctx.Error(errs.NewBadRequestError("Address ID is required",
			fmt.Errorf("missing address ID in request for user %s", userEntity.ID)))
		return
	}

	if ctx.Request.Body != nil && ctx.Request.ContentLength > 0 {
		ctx.Error(errs.NewBadRequestError("Invalid Request for Address Deletion",
			fmt.Errorf("DELETE request with body received.")))
		return
	}

	err := h.userService.DeleteUserAddress(ctx, userEntity.ID, addressID)
	if err != nil {
		if err.Error() == "user not found" {
			ctx.Error(errs.NewNotFoundError("User not found",
				fmt.Errorf("no user found with ID: %s", userEntity.ID)))
			return
		}
		if err.Error() == "address not found" {
			ctx.Error(errs.NewNotFoundError("Address not found",
				fmt.Errorf("no address found with ID: %s for user %s", addressID, userEntity.ID)))
			return
		}
		ctx.Error(errs.NewInternalServerError("Failed to delete address",
			fmt.Errorf("error deleting address %s for user %s: %w", addressID, userEntity.ID, err)))
		return
	}

	ctx.JSON(http.StatusOK, "Address deleted successfully")
}
