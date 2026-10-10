package vendors

import (
	"context"
	

	"cloud.google.com/go/pubsub"
	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	"github.com/sirupsen/logrus"
	
)

func InitPubSub() *pubsub.Client {
    config := config.GetConfig()
    ctx := context.Background()
 
    // No need to check credentials file – use default credentials
    client, err := pubsub.NewClient(ctx, config.PubSubConfig.ProjectID)
    if err != nil {
        logrus.Fatalf("Failed to create pubsub client: %v", err)
    }
    logrus.Info("PubSub client created successfully using default credentials")
    return client
}