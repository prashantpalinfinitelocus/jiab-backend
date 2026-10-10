package vendors

import (
	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
)

type InfobipClient struct {
	ApiClient      *APIClient
	MediaApiClient *APIClient
	Cfg            config.InfobipConfig
}

func InitInfobip() *InfobipClient {
	cfg := config.GetConfig()
	apiClient := InitializeClient(cfg.InfobipConfig.BaseURL, map[string]string{"Authorization": "App " + cfg.InfobipConfig.APIKey, "Content-Type": "application/json"})
	mediaApiClient := InitializeClient(cfg.InfobipConfig.BaseURL, map[string]string{"Authorization": "App " + cfg.InfobipConfig.MediaAPIKey, "Content-Type": "application/json"})
	return &InfobipClient{ApiClient: apiClient, MediaApiClient: mediaApiClient, Cfg: cfg.InfobipConfig}
}
