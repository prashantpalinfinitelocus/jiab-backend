package dtos

type WhatsAppTemplateRequest struct {
	To           string   `json:"to" binding:"required"`
	MessageID    string   `json:"message_id" binding:"required"`
	Placeholders []string `json:"placeholders" binding:"required"`
}

type WhatsAppRejectedTemplateRequest struct {
	To        string `json:"to" binding:"required"`
	MessageID string `json:"message_id" binding:"required"`
}

type WhatsAppPlagiarizedTemplateRequest struct {
	To        string `json:"to" binding:"required"`
	MessageID string `json:"message_id" binding:"required"`
}

type WhatsAppTemplateResponse struct {
	Messages []WhatsAppMessage `json:"messages"`
}

type WhatsAppMessage struct {
	MessageID string `json:"messageId"`
	Status    struct {
		GroupID     int    `json:"groupId"`
		GroupName   string `json:"groupName"`
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"status"`
	To string `json:"to"`
}

type WhatsAppTemplateBody struct {
	Messages []WhatsAppTemplateMessage `json:"messages"`
}

type WhatsAppTemplateMessage struct {
	From      string                  `json:"from"`
	To        string                  `json:"to"`
	MessageID string                  `json:"messageId"`
	Content   WhatsAppTemplateContent `json:"content"`
}

type WhatsAppTemplateContent struct {
	TemplateName string               `json:"templateName"`
	TemplateData WhatsAppTemplateData `json:"templateData"`
	Language     string               `json:"language"`
}

type WhatsAppTemplateData struct {
	Body    WhatsAppTemplateBodyData `json:"body"`
	Header  *WhatsAppTemplateHeader  `json:"header,omitempty"`
	Buttons []WhatsAppTemplateButton `json:"buttons,omitempty"`
}

type WhatsAppTemplateBodyData struct {
	Placeholders []string `json:"placeholders"`
}

type WhatsAppTemplateHeader struct {
	Type     string `json:"type"`
	MediaURL string `json:"mediaUrl"`
}

type WhatsAppTemplateButton struct {
	Type      string `json:"type"`
	Parameter string `json:"parameter"`
}
