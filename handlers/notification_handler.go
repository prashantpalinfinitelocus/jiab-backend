package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	_ "github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type NotificationHandler interface {
	RegisterDevice(c *gin.Context)
	GetNotifications(c *gin.Context)
	GetNotificationCount(c *gin.Context)
	BroadcastNotification(c *gin.Context)
	SendNotificationByPhone(c *gin.Context)
	MarkAsRead(c *gin.Context)
}

type notificationHandler struct {
	notificationService services.NotificationService
}

func NewNotificationHandler(notificationService services.NotificationService) NotificationHandler {
	return &notificationHandler{
		notificationService: notificationService,
	}
}

// @Summary Register a device for push notifications
// @Description Register a device for push notifications
// @Tags Notifications
// @Accept json
// @Produce json
// @Param request body dtos.RegisterDeviceDTO true "Device token"
// @Success 200 {object} utils.ResponseType[string] "Success"
// @Failure 400 {object} map[string]string "Bad Request"
// @Security BearerAuth
// @Router /notifications/register-device [post]
func (h *notificationHandler) RegisterDevice(c *gin.Context) {
	var registerDeviceDTO dtos.RegisterDeviceDTO
	if err := c.ShouldBindJSON(&registerDeviceDTO); err != nil {
		c.Error(errs.NewBadRequestError("invalid request body", err))
		return
	}

	user, _ := c.Get("user")
	userID := user.(*entities.User).ID

	err := h.notificationService.RegisterDevice(c.Request.Context(), userID, registerDeviceDTO.DeviceToken)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device registered successfully"})
}

// @Summary Get notifications
// @Description Get notifications
// @Tags Notifications
// @Param page_number query int false "Page number"
// @Param page_size query int false "Page size"
// @Accept json
// @Produce json
// @Success 200 {object} utils.ResponseType[dtos.NotificationResponse] "Success"
// @Failure 400 {object} map[string]string "Bad Request"
// @Security BearerAuth
// @Router /notifications [get]
func (h *notificationHandler) GetNotifications(c *gin.Context) {
	var notificationFilter dtos.NotificationFilter
	if err := c.ShouldBindQuery(&notificationFilter); err != nil {
		c.Error(errs.NewBadRequestError("invalid request body", err))
		return
	}

	user, _ := c.Get("user")
	userID := user.(*entities.User).ID
	notifications, totalCount, err := h.notificationService.GetUserNotifications(c.Request.Context(), userID, notificationFilter)

	notificationResponse := dtos.NotificationResponse{
		Notifications: notifications,
		TotalCount:    totalCount,
	}

	if notificationFilter.PageNumber > 0 && notificationFilter.PageSize > 0 {
		notificationResponse.PageNumber = notificationFilter.PageNumber
		notificationResponse.PageSize = notificationFilter.PageSize
	}

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, notificationResponse)
}

// @Summary Get notification count
// @Description Get notification count
// @Tags Notifications
// @Accept json
// @Produce json
// @Success 200 {object} utils.ResponseType[dtos.NotificationCountResponse] "Success"
// @Failure 400 {object} map[string]string "Bad Request"
// @Security BearerAuth
// @Router /notifications/count [get]
func (h *notificationHandler) GetNotificationCount(c *gin.Context) {
	user, _ := c.Get("user")
	userID := user.(*entities.User).ID
	count, err := h.notificationService.GetNotificationCount(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dtos.NotificationCountResponse{
		Count: count,
	})
}

// @Summary Broadcast notification
// @Description Broadcast a push notification to every active FCM device token
// @Tags Notifications
// @Accept json
// @Produce json
// @Param request body dtos.BroadcastNotificationDTO true "Broadcast notification request"
// @Success 200 {object} utils.ResponseType[string] "Success"
// @Security ApiKeyAuth
// @Failure 400 {object} map[string]string "Bad Request"
// @Router /notifications/broadcast [post]
func (h *notificationHandler) BroadcastNotification(c *gin.Context) {
	var req dtos.BroadcastNotificationDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	// Detached from request cancellation for the same reason as the campaign
	// endpoints: a broadcast to every active device token can outlast the
	// client's connection/timeout, and must not be aborted by that.
	err := h.notificationService.BroadcastNotification(context.WithoutCancel(c.Request.Context()), req.Title, req.Body, req.ImageURL, req.BaseURL)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification sent"})
}

// @Summary Send notification by phone number
// @Description Send a push notification to a specific user by their phone number
// @Tags Notifications
// @Accept json
// @Produce json
// @Param request body dtos.PhoneNotificationDTO true "Phone notification request"
// @Success 200 {object} utils.ResponseType[string] "Success"
// @Security ApiKeyAuth
// @Failure 400 {object} map[string]string "Bad Request"
// @Router /notifications/send-to-phone [post]
func (h *notificationHandler) SendNotificationByPhone(c *gin.Context) {
	var req dtos.PhoneNotificationDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	err := h.notificationService.SendNotificationByPhone(context.WithoutCancel(c.Request.Context()), req.PhoneNumber, req.Title, req.Body, req.ImageURL)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification sent to user"})
}

// @Summary Mark as read
// @Description Mark as read
// @Tags Notifications
// @Accept json
// @Produce json
// @Success 200 {object} utils.ResponseType[string] "Success"
// @Failure 400 {object} map[string]string "Bad Request"
// @Security BearerAuth
// @Router /notifications/mark-as-read [post]
func (h *notificationHandler) MarkAsRead(c *gin.Context) {
	user, _ := c.Get("user")
	userID := user.(*entities.User).ID

	err := h.notificationService.MarkAsRead(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notifications marked as read"})
}
