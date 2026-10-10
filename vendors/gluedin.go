package vendors

import (
	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
)

func InitGluedin() (*APIClient, string) {
	cfg := config.GetConfig()
	apiClient := InitializeClient(cfg.GluedinConfig.BaseURL, map[string]string{"x-api-key": cfg.GluedinConfig.APIKey})
	return apiClient, cfg.GluedinConfig.ProjectID
}
