package services

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

type CDPService interface {
	SendEvent(ctx context.Context, event dtos.CDPEventRequest) (*dtos.CDPEventResponse, error)
}

type cdpService struct {
	cdpClient *vendors.APIClient
	config    config.CDPConfig
}

func NewCDPService(cdpClient *vendors.APIClient, config config.CDPConfig) CDPService {
	return &cdpService{
		cdpClient: cdpClient,
		config:    config,
	}
}

func (s *cdpService) buildCDPPayload(event dtos.CDPEventRequest) map[string]interface{} {
	payload := make(map[string]interface{})

	payload["client_id"] = s.config.ClientID
	payload["event_type"] = event.EventType
	payload["event_sub_type"] = event.EventSubType
	payload["brand_name"] = event.BrandName
	payload["phone_e164"] = event.PhoneE164

	if event.Email != nil {
		payload["email"] = *event.Email
	}
	if event.FirstName != nil {
		payload["first_name"] = *event.FirstName
	}
	if event.TargetingAgeFrom != nil {
		payload["targeting_age_from"] = *event.TargetingAgeFrom
	}
	if event.GeoCountryCode != nil {
		payload["geo_country_code"] = *event.GeoCountryCode
	}
	if event.GeoStateProvinceCode != nil {
		payload["geo_state_province_code"] = *event.GeoStateProvinceCode
	}
	if event.GeoCityName != nil {
		payload["geo_city_name"] = *event.GeoCityName
	}
	if event.GeoPostalCode != nil {
		payload["geo_postal_code"] = *event.GeoPostalCode
	}
	if event.IPAddress != nil {
		payload["ip_address"] = *event.IPAddress
	}
	if event.Referrer != nil {
		payload["referrer"] = *event.Referrer
	}
	if event.LanguageCode != nil {
		payload["language_code"] = *event.LanguageCode
	}
	if event.AddressLine1 != nil {
		payload["address_line1"] = *event.AddressLine1
	}
	if event.AddressLine2 != nil {
		payload["address_line2"] = *event.AddressLine2
	}
	if event.AddressCity != nil {
		payload["address_city"] = *event.AddressCity
	}
	if event.AddressState != nil {
		payload["address_state"] = *event.AddressState
	}
	if event.DOB != nil {
		payload["dob"] = *event.DOB
	}
	if event.Gender != nil {
		payload["gender"] = *event.Gender
	}

	if event.SMSMessage != nil {
		payload["communication_preferences/sms_message"] = *event.SMSMessage
	}
	if event.PhoneCall != nil {
		payload["communication_preferences/phone_call"] = *event.PhoneCall
	}
	if event.EmailPref != nil {
		payload["communication_preferences/email"] = *event.EmailPref
	}
	if event.WhatsApp != nil {
		payload["communication_preferences/whatsapp"] = *event.WhatsApp
	}
	if event.CommunicationPreferences != nil {
		for key, value := range event.CommunicationPreferences {
			payload["communication_preferences/"+key] = value
		}
	}

	if event.AdditionalData != nil {
		for key, value := range event.AdditionalData {
			payload[key] = value
		}
	}

	return payload
}

func (s *cdpService) SendEvent(ctx context.Context, event dtos.CDPEventRequest) (*dtos.CDPEventResponse, error) {
	logrus.WithFields(logrus.Fields{
		"event_type":     utils.SanitizeForLog(event.EventType),
		"event_sub_type": utils.SanitizeForLog(event.EventSubType),
		"brand_name":     utils.SanitizeForLog(event.BrandName),
		"phone_e164":     utils.SanitizeForLog(event.PhoneE164),
	}).Info("Sending CDP event")

	payload := s.buildCDPPayload(event)
	logrus.WithField("payload", utils.SanitizeForLog(fmt.Sprintf("%+v", payload))).Info("CDP payload")

	response, err := s.cdpClient.Post(constants.CDP_EVENT_ENDPOINT, payload, nil, false)
	if err != nil {
		logrus.WithError(err).Error("Failed to send CDP event")
		return nil, fmt.Errorf("failed to send CDP event: %w", err)
	}

	if !response.IsSuccess() {
		logrus.WithFields(logrus.Fields{
			"status_code": response.StatusCode,
			"response":    response.GetBodyAsString(),
		}).Error("CDP API returned non-success status")
		return nil, errors.NewInternalServerError("CDP API request failed", fmt.Errorf("status code: %d", response.StatusCode))
	}

	var cdpResponse dtos.CDPEventResponse
	if err := response.ParseJSON(&cdpResponse); err != nil {
		logrus.WithError(err).Error("Failed to parse CDP response")
		return nil, fmt.Errorf("failed to parse CDP response: %w", err)
	}

	logrus.WithField("status", cdpResponse.Status).Info("CDP event sent successfully")
	return &cdpResponse, nil
}
