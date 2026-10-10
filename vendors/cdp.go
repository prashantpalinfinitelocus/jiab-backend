package vendors

import (
	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
)

func InitCDP() *APIClient {
	cfg := config.GetConfig()
	headers := map[string]string{
		"accept":       "application/json",
		"x-api-key":    cfg.CDPConfig.APIKey,
		"Content-Type": "application/json",
	}

	return InitializeClient(cfg.CDPConfig.BaseURL, headers)
}
