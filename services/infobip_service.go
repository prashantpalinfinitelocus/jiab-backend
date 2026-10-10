package services

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

type InfobipService interface {
	SendOTP(phoneNumber string, otp string) error
	SendReferral(phoneNumber string, referralCode string) error
	FetchMedia(ctx context.Context, mediaId string) (io.ReadCloser, error)
	SendWhatsAppSuccessTemplate(request dtos.WhatsAppTemplateRequest) error
	SendWhatsAppRejectedTemplate(request dtos.WhatsAppRejectedTemplateRequest) error
	SendWhatsAppPlagiarizedTemplate(request dtos.WhatsAppPlagiarizedTemplateRequest) error
}

type infobipService struct {
	infobipClient *vendors.InfobipClient
}

func NewInfobipService(infobipClient *vendors.InfobipClient) InfobipService {
	return &infobipService{infobipClient: infobipClient}
}

func (s *infobipService) SendOTP(phoneNumber string, otp string) error {
	body := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"from": "SPRITE",
				"destinations": []map[string]interface{}{
					{
						"to": fmt.Sprintf("91%s", phoneNumber),
					},
				},
				"content": map[string]interface{}{
					"text": fmt.Sprintf("Please use the One Time Password %s to login on Sprite Promotion Website. Please do not share it with anyone - SPRITE", otp),
				},
				"options": map[string]interface{}{
					"regional": map[string]interface{}{
						"indiaDlt": map[string]interface{}{
							"principalEntityId": "1101393580000056521",
							"contentTemplateId": "1107172732827908608",
						},
					},
				},
			},
		},
	}
	_, err := s.infobipClient.ApiClient.Post("/sms/3/messages", body, nil, false)
	if err != nil {
		logrus.Error("error sending OTP", err)
		return err
	}

	return nil
}

func (s *infobipService) SendReferral(phoneNumber string, referralCode string) error {
	body := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"from": "SPRITE",
				"destinations": []map[string]interface{}{
					{
						"to": fmt.Sprintf("91%s", phoneNumber),
					},
				},
				"content": map[string]interface{}{
					"text": fmt.Sprintf("Hi User, laughter is best when shared! Join Sprite Joke-In-A-Bottle using your friend's code %s. Check T&Cs: sprite-joke-in-a-bottle.coke2home.com", referralCode),
				},
				"options": map[string]interface{}{
					"regional": map[string]interface{}{
						"indiaDlt": map[string]interface{}{
							"principalEntityId": "1101393580000056521",
							"contentTemplateId": "1107175179828005661",
						},
					},
				},
			},
		},
	}
	_, err := s.infobipClient.ApiClient.Post("/sms/3/messages", body, nil, false)
	if err != nil {
		logrus.Error("error sending referral", err)
		return err
	}

	return nil
}

func (s *infobipService) FetchMedia(ctx context.Context, mediaId string) (io.ReadCloser, error) {
	response, err := s.infobipClient.MediaApiClient.Get(fmt.Sprintf("/whatsapp/1/senders/%s/media/%s", s.infobipClient.Cfg.WANumber, mediaId), nil)
	if err != nil {
		return nil, errs.NewInternalServerError("error fetching media", err)
	}
	return io.NopCloser(bytes.NewReader(response.Body)), nil
}

func (s *infobipService) SendWhatsAppSuccessTemplate(request dtos.WhatsAppTemplateRequest) error {
	templateBody := dtos.WhatsAppTemplateBody{
		Messages: []dtos.WhatsAppTemplateMessage{
			{
				From:      constants.WA_FROM_NUMBER,
				To:        request.To,
				MessageID: request.MessageID,
				Content: dtos.WhatsAppTemplateContent{
					TemplateName: constants.WA_UGC_SUCCESS,
					TemplateData: dtos.WhatsAppTemplateData{
						Body: dtos.WhatsAppTemplateBodyData{
							Placeholders: request.Placeholders,
						},
						Header: &dtos.WhatsAppTemplateHeader{
							Type:     "IMAGE",
							MediaURL: constants.WA_PJ_CHALLENGE_WINNER_IMAGE_URL,
						},
					},
					Language: "en_GB",
				},
			},
		},
	}

	_, err := s.infobipClient.MediaApiClient.Post(constants.WA_TEMPLATE_ENDPOINT, templateBody, nil, false)
	if err != nil {
		logrus.Error("error sending WhatsApp template message", err)
		return err
	}

	return nil
}

func (s *infobipService) SendWhatsAppRejectedTemplate(request dtos.WhatsAppRejectedTemplateRequest) error {
	templateBody := dtos.WhatsAppTemplateBody{
		Messages: []dtos.WhatsAppTemplateMessage{
			{
				From:      constants.WA_FROM_NUMBER,
				To:        request.To,
				MessageID: request.MessageID,
				Content: dtos.WhatsAppTemplateContent{
					TemplateName: constants.WA_UGC_REJECTED_TEMPLATE_NAME,
					TemplateData: dtos.WhatsAppTemplateData{
						Body: dtos.WhatsAppTemplateBodyData{
							Placeholders: []string{},
						},
						Buttons: []dtos.WhatsAppTemplateButton{
							{
								Type:      "QUICK_REPLY",
								Parameter: "Submit Another Joke",
							},
							{
								Type:      "QUICK_REPLY",
								Parameter: "Get More Jokes",
							},
							{
								Type:      "QUICK_REPLY",
								Parameter: "Change Joke Language",
							},
						},
					},
					Language: "en_GB",
				},
			},
		},
	}

	_, err := s.infobipClient.MediaApiClient.Post(constants.WA_TEMPLATE_ENDPOINT, templateBody, nil, false)
	if err != nil {
		logrus.Error("error sending WhatsApp rejected template message", err)
		return err
	}

	return nil
}

func (s *infobipService) SendWhatsAppPlagiarizedTemplate(request dtos.WhatsAppPlagiarizedTemplateRequest) error {
	templateBody := dtos.WhatsAppTemplateBody{
		Messages: []dtos.WhatsAppTemplateMessage{
			{
				From:      constants.WA_FROM_NUMBER,
				To:        request.To,
				MessageID: request.MessageID,
				Content: dtos.WhatsAppTemplateContent{
					TemplateName: constants.WA_UGC_PLAGIARIZED_TEMPLATE_NAME,
					TemplateData: dtos.WhatsAppTemplateData{
						Body: dtos.WhatsAppTemplateBodyData{
							Placeholders: []string{},
						},
						Buttons: []dtos.WhatsAppTemplateButton{
							{
								Type:      "QUICK_REPLY",
								Parameter: "Submit Another Joke",
							},
							{
								Type:      "QUICK_REPLY",
								Parameter: "Get More Jokes",
							},
							{
								Type:      "QUICK_REPLY",
								Parameter: "Change Joke Language",
							},
						},
					},
					Language: "en_GB",
				},
			},
		},
	}

	_, err := s.infobipClient.MediaApiClient.Post(constants.WA_TEMPLATE_ENDPOINT, templateBody, nil, false)
	if err != nil {
		logrus.Error("error sending WhatsApp plagiarized template message", err)
		return err
	}

	return nil
}
