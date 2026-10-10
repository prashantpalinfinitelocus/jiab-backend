package vendors

import (
	"context"
	"fmt"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
)

type CustomerDataServiceClient struct {
	DatamartClient    *APIClient
	HashedKOCIDClient *APIClient
	Config            config.CustomerDataServiceConfig
}

func InitCustomerDataService() *CustomerDataServiceClient {
	cfg := config.GetConfig().CustomerDataServiceConfig

	headers := map[string]string{
		"x-api-key":    cfg.APIKey,
		"Content-Type": "application/json",
	}

	datamartClient := InitializeClient(cfg.DatamartBaseURL, headers)
	datamartClient.SetTimeout(time.Duration(cfg.Timeout) * time.Second)

	kocidClient := InitializeClient(cfg.HashedKOCIDBaseURL, headers)
	kocidClient.SetTimeout(time.Duration(cfg.Timeout) * time.Second)

	return &CustomerDataServiceClient{
		DatamartClient:    datamartClient,
		HashedKOCIDClient: kocidClient,
		Config:            cfg,
	}
}

func (c *CustomerDataServiceClient) CollectUserData(ctx context.Context, req *dtos.DatamartCollectRequest) error {
	resp, err := c.DatamartClient.Post("/v2/datamart/collect", req, nil, true)
	if err != nil {
		return fmt.Errorf("datamart collect failed: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("datamart collect failed with status %d: %s", resp.StatusCode, resp.GetBodyAsString())
	}

	var collectResp dtos.DatamartCollectResponse
	if err := resp.ParseJSON(&collectResp); err != nil {
		return fmt.Errorf("failed to parse datamart response: %w", err)
	}

	if collectResp.Status != 0 {
		return fmt.Errorf("datamart collect failed: %s", collectResp.Error)
	}

	return nil
}

func (c *CustomerDataServiceClient) GetHashedKOCID(ctx context.Context, req *dtos.GetHashedKOCIDRequest) (string, error) {
	resp, err := c.HashedKOCIDClient.Post("/v4/datamart/getmemberhashedkocid", req, nil, true)
	if err != nil {
		return "", fmt.Errorf("get hashed KOCID failed: %w", err)
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("get hashed KOCID failed with status %d: %s", resp.StatusCode, resp.GetBodyAsString())
	}

	var kocidResp dtos.GetHashedKOCIDResponse
	if err := resp.ParseJSON(&kocidResp); err != nil {
		return "", fmt.Errorf("failed to parse KOCID response: %w", err)
	}

	if kocidResp.Status != 0 {
		return "", fmt.Errorf("get hashed KOCID failed: %s", kocidResp.Error)
	}

	if kocidResp.Data == nil || kocidResp.Data.KOIdentifier == "" {
		return "", fmt.Errorf("empty KOCID returned")
	}

	return kocidResp.Data.KOIdentifier, nil
}
