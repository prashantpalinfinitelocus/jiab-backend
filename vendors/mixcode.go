package vendors

import (
	"encoding/base64"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
)

type MixcodeClient struct {
	ApiClient *APIClient
	Cfg       config.Mixcode
}

func InitMixcode() *MixcodeClient {
	cfg := config.GetConfig()
	authToken := base64.StdEncoding.EncodeToString([]byte(cfg.Mixcode.ProgramID + ":" + cfg.Mixcode.ProgramSecret))
	apiClient := InitializeClient(cfg.Mixcode.BaseURL, map[string]string{"Accept": "application/json", "Authorization": "Basic " + authToken})
	return &MixcodeClient{ApiClient: apiClient, Cfg: cfg.Mixcode}
}
