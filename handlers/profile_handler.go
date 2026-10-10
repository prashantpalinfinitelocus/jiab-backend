package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type ProfileService interface {
	GetProfile(ctx *gin.Context)
	EditProfile(ctx *gin.Context)
	WAUserDetails(ctx *gin.Context)
	WAEditLastLanguage(ctx *gin.Context)
	ChangeChatLanguage(ctx *gin.Context)
}

type ProfileHandler struct {
	userService      services.UserService
	comicCoinService services.ComicCoinService
}

func NewProfileHandler(userService services.UserService, comicCoinService services.ComicCoinService) *ProfileHandler {
	return &ProfileHandler{
		userService:      userService,
		comicCoinService: comicCoinService,
	}
}

// GetProfile godoc
// @Summary Get Profile
// @Description Get the user's profile information including rank and balance
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.ResponseType[dtos.ProfileResponseDTO] "Profile retrieved successfully"
// @Router /profile [get]
func (h *ProfileHandler) GetProfile(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)
	userProfile, err := h.userService.GetUser(ctx, userEntity.ID)
	if err != nil {
		ctx.Error(errs.NewInternalServerError("Failed to fetch user profile", fmt.Errorf("error fetching user profile: %w", err)))
		return
	}

	if userProfile == nil {
		ctx.Error(errs.NewNotFoundError("User not found", fmt.Errorf("no user found with ID: %s", userEntity.ID)))
		return
	}

	// Get user rank and balance
	userRank, err := h.userService.GetUserRank(ctx, userEntity.ID)
	if err != nil {
		ctx.Error(errs.NewInternalServerError("Failed to fetch user rank", fmt.Errorf("error fetching user rank: %w", err)))
		return
	}

	wallet, err := h.comicCoinService.GetUserWalletWithoutTx(ctx, userEntity.ID)
	if err != nil {
		ctx.Error(errs.NewInternalServerError("Failed to fetch user wallet", fmt.Errorf("error fetching user wallet: %w", err)))
		return
	}

	response := dtos.ProfileResponseDTO{
		User:           *userProfile,
		Rank:           nil,
		CurrentBalance: 0,
	}

	if userRank != nil {
		response.Rank = &userRank.UserRank
	}

	if wallet != nil {
		response.CurrentBalance = wallet.CurrentBalance
	}

	ctx.JSON(http.StatusOK, response)
}

// EditProfile godoc
// @Summary Edit Profile
// @Description Edit the user's profile information
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dtos.EditProfileRequestDTO true "Edit Profile Request"
// @Success 200 {object} utils.ResponseType[entities.User] "Profile updated successfully"
// @Router /profile [put]
func (h *ProfileHandler) EditProfile(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.Error(errs.NewUnauthorizedError("User not authenticated", fmt.Errorf("no user found in context")))
		return
	}

	userEntity := user.(*entities.User)

	var req dtos.EditProfileRequestDTO
	if err := ctx.ShouldBind(&req); err != nil {
		logrus.Errorf("validation error: %v", err)
		validationErrors := utils.FormatValidationErrors(err)
		ctx.Error(errs.NewValidationErrorWithFields("Invalid request body", validationErrors))
		return
	}

	// Convert form-data string values to appropriate types
	if avatarIDStr := ctx.PostForm("avatar_id"); avatarIDStr != "" {
		avatarID, err := strconv.Atoi(avatarIDStr)
		if err != nil {
			ctx.Error(errs.NewBadRequestError("Invalid avatar_id format",
				fmt.Errorf("avatar_id must be a number: %w", err)))
			return
		}
		req.AvatarID = &avatarID
	}

	if isAvatarStr := ctx.PostForm("is_avatar"); isAvatarStr != "" {
		isAvatar, err := strconv.ParseBool(isAvatarStr)
		if err != nil {
			ctx.Error(errs.NewBadRequestError("Invalid is_avatar format",
				fmt.Errorf("is_avatar must be true or false: %w", err)))
			return
		}
		req.IsAvatar = &isAvatar
	}

	if isProfileStr := ctx.PostForm("is_profile"); isProfileStr != "" {
		isProfile, err := strconv.ParseBool(isProfileStr)
		if err != nil {
			ctx.Error(errs.NewBadRequestError("Invalid is_profile format",
				fmt.Errorf("is_profile must be true or false: %w", err)))
			return
		}
		req.IsProfile = &isProfile
	}

	// Validate avatar requirements first
	if req.IsAvatar != nil && *req.IsAvatar && req.AvatarID == nil {
		ctx.Error(errs.NewBadRequestError("avatar_id is required when is_avatar is true",
			fmt.Errorf("avatar_id missing for user %s when is_avatar is true", userEntity.ID)))
		return
	}

	// Validate that at least one field is provided
	if req.Name == nil && req.Email == nil && req.Gender == nil && req.DOB == nil &&
		(req.AvatarID == nil || (req.IsAvatar != nil && !*req.IsAvatar)) &&
		(req.ProfileImage == nil || (req.IsProfile != nil && !*req.IsProfile)) {
		ctx.Error(errs.NewBadRequestError("At least one field must be provided for update",
			fmt.Errorf("no fields provided in update request for user: %s", userEntity.ID)))
		return
	}

	// Validate email format if provided
	if req.Email != nil {
		if !utils.IsValidEmail(*req.Email) {
			ctx.Error(errs.NewBadRequestError("Invalid email format",
				fmt.Errorf("invalid email format provided: %s", *req.Email)))
			return
		}
	}

	// Validate gender if provided
	if req.Gender != nil {
		validGenders := map[string]bool{
			"male":              true,
			"female":            true,
			"other":             true,
			"perfer_not_to_say": true,
		}
		if !validGenders[*req.Gender] {
			ctx.Error(errs.NewBadRequestError("Invalid gender value. Must be one of: male, female, other",
				fmt.Errorf("invalid gender value provided: %s", *req.Gender)))
			return
		}
	}

	if req.DOB != nil {
		if *req.DOB == "" {
			req.DOB = nil
		} else {
			if err := h.validateAge(*req.DOB, userEntity.ID); err != nil {
				ctx.Error(err)
				return
			}
		}
	}

	// Update the profile
	updatedUser, err := h.userService.EditProfile(ctx, userEntity.ID, req)
	if err != nil {
		switch {
		case err.Error() == "user not found":
			ctx.Error(errs.NewNotFoundError("User not found",
				fmt.Errorf("no user found with ID: %s", userEntity.ID)))
		case strings.Contains(err.Error(), "avatar with ID"):
			ctx.Error(errs.NewNotFoundError(err.Error(),
				fmt.Errorf("avatar validation error for user %s: %w", userEntity.ID, err)))
		case strings.Contains(err.Error(), "email already in use"):
			ctx.Error(errs.NewBadRequestError("Email already in use",
				fmt.Errorf("email validation error for user %s: %w", userEntity.ID, err)))
		default:
			ctx.Error(errs.NewInternalServerError("Failed to update profile",
				fmt.Errorf("error updating profile for user %s: %w", userEntity.ID, err)))
		}
		return
	}

	ctx.JSON(http.StatusOK, *updatedUser)
}

func (h *ProfileHandler) WAUserDetails(ctx *gin.Context) {
	// Get UserId from the URL query
	queryUserID := ctx.Query("UserId")
	if queryUserID == "" {
		ctx.JSON(400, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "UserId is required in query",
			"succeeded":       false,
			"statusCode":      400,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}

	// Get claims from the context (set by the GeneralAuth middleware)
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(401, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "Unauthorized",
			"succeeded":       false,
			"statusCode":      401,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}

	// Proceed to fetch user details using the verified user ID
	resp, err := h.userService.WAUserDetails(ctx, user.(*entities.User).ID)
	if err != nil {
		// Handle all errors by extracting message and status from service layer
		if appErr, ok := err.(*errs.AppError); ok {
			ctx.JSON(appErr.Status, gin.H{
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
		ctx.JSON(500, gin.H{
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

	// Return the response data directly without wrapping in a "data" key
	ctx.JSON(200, resp)
}

func (h *ProfileHandler) WAEditLastLanguage(ctx *gin.Context) {
	languageIDStr := ctx.Query("LanguageId")
	if languageIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "LanguageId is required",
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

	languageID, err := strconv.Atoi(languageIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "Invalid LanguageId",
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

	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"data":            nil,
			"failed":          true,
			"message":         "User not authenticated",
			"succeeded":       false,
			"statusCode":      http.StatusUnauthorized,
			"page":            0,
			"totalPages":      0,
			"totalCount":      0,
			"hasPreviousPage": false,
			"hasNextPage":     false,
		})
		return
	}
	userEntity := user.(*entities.User)

	if err := h.userService.EditLastLanguage(ctx, userEntity.ID, languageID); err != nil {
		// Handle all errors by extracting message and status from service layer
		if appErr, ok := err.(*errs.AppError); ok {
			ctx.JSON(appErr.Status, gin.H{
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
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"data":            nil,
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

	ctx.JSON(http.StatusOK, gin.H{
		"data":            "Last Language Selection updated successfully.",
		"failed":          false,
		"message":         "Success",
		"succeeded":       true,
		"statusCode":      http.StatusOK,
		"page":            0,
		"totalPages":      0,
		"totalCount":      0,
		"hasPreviousPage": false,
		"hasNextPage":     false,
	})
}

// ChangeChatLanguage godoc
// @Summary Change Chat Language
// @Description Change the user's chat language
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param languageId query int true "Language ID"
// @Success 200 {object} utils.ResponseType[string] "Last Language Selection updated successfully"
// @Router /profile/change-chat-language [post]
func (h *ProfileHandler) ChangeChatLanguage(ctx *gin.Context) {
	user, _ := ctx.Get("user")
	userEntity := user.(*entities.User)

	languageIDStr := ctx.Query("languageId")
	if languageIDStr == "" {
		ctx.Error(errs.NewBadRequestError("LanguageId is required", fmt.Errorf("languageId is required")))
		return
	}

	languageID, err := strconv.Atoi(languageIDStr)
	if err != nil {
		ctx.Error(errs.NewBadRequestError("Invalid languageId", fmt.Errorf("languageId must be a number: %w", err)))
		return
	}

	if err := h.userService.EditLastLanguage(ctx, userEntity.ID, languageID); err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Last Language Selection updated successfully.",
	})
}

// UpdateProfileComicCoinPopUp godoc
// @Summary Update Profile Comic Coin Pop Up
// @Description Update the user's profile comic coin pop up
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.ResponseType[string] "Profile comic coin pop up updated successfully"
// @Router /profile/comic-coin-pop-up [patch]
func (h *ProfileHandler) UpdateProfileComicCoinPopUp(ctx *gin.Context) {
	user, _ := ctx.Get("user")

	userEntity := user.(*entities.User)
	if err := h.userService.UpdateProfileComicCoinPopUp(ctx, userEntity.ID); err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Profile comic coin pop up updated successfully.",
	})
}


func (h *ProfileHandler) validateAge(dobStr string, userID string) error {
	if dobStr == "" {
		return nil
	}

	dob, err := time.Parse("01/02/2006", dobStr)
	if err != nil {
		return errs.NewBadRequestError("Invalid date format. Please use MM/DD/YYYY format or leave it empty",
			fmt.Errorf("invalid date format provided: %s, error: %w", dobStr, err))
	}

	now := time.Now()
	age := now.Year() - dob.Year()

	if now.YearDay() < dob.YearDay() {
		age--
	}

	if age < constants.MIN_AGE_REQUIRED {
		return errs.NewBadRequestError("User must be at least 18 years old",
			fmt.Errorf("user age %d is below minimum required age of %d for user %s", age, constants.MIN_AGE_REQUIRED, userID))
	}

	if age >= constants.MAX_AGE_ALLOWED {
		return errs.NewBadRequestError("Invalid age. Age must be less than 150 years",
			fmt.Errorf("user age %d exceeds maximum allowed age of %d for user %s", age, constants.MAX_AGE_ALLOWED-1, userID))
	}

	if dob.After(now) {
		return errs.NewBadRequestError("Date of birth cannot be in the future",
			fmt.Errorf("future date of birth provided: %s for user %s", dobStr, userID))
	}

	return nil
}