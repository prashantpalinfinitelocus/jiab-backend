package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config struct represents the application configuration
type Config struct {
	AppEnv                    string
	AppPort                   string
	DbConfig                  DatabaseConfig
	InfobipConfig             InfobipConfig
	JwtConfig                 JwtConfig
	GluedinConfig             GluedinConfig
	SSHConfig                 SSHConfig
	GcsConfig                 GcsConfig
	FirebaseConfig            FirebaseConfig
	KafkaConfig               KafkaConfig
	APIKey                    string
	XAPIKey                   string
	Mixcode                   Mixcode
	PubSubConfig              PubSubConfig
	ChatbotGluedinUserID      string
	CDPConfig                 CDPConfig
	CustomerDataServiceConfig CustomerDataServiceConfig
	FrontendBaseURL           string
	MemeGeneratorConfig       MemeGeneratorConfig
}

// MemeGeneratorConfig holds configuration for meme generator
type MemeGeneratorConfig struct {
	FontPath         string
	BackdropURL      string
	LogoURL          string
	PythonServiceURL string
}

// Singleton instance of Config
var (
	conf *Config
	once sync.Once
)

// DatabaseConfig struct for database-related configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type SSHConfig struct {
	UseSSH        string
	Host          string
	User          string
	Password      string
	RDSEndpoint   string
	RDSPort       int
	KafkaEndpoint []string
	KafkaPort     int
}

// OAuthConfig struct for OAuth-related configuration
type InfobipConfig struct {
	BaseURL     string
	APIKey      string
	MediaAPIKey string
	WANumber    string
}

// JwtConfig struct for JWT-related configuration
type JwtConfig struct {
	SecretKey          string
	AccessTokenExpiry  int
	RefreshTokenExpiry int
	TempTokenExpiry    int
}

type GluedinConfig struct {
	BaseURL     string
	APIKey      string
	ProjectID   string
	UGCEndpoint string
}

// S3Config struct for AWS S3 configuration
type S3Config struct {
	BucketName string
	Region     string
	AccessKey  string
	SecretKey  string
	OldS3Url   string
}

// GcsConfig struct for GCP configuration
type GcsConfig struct {
	BucketName string
	ProjectID  string
	GcpUrl     string
}

// FirebaseConfig struct for Firebase-related configuration
type FirebaseConfig struct {
	ServiceKeyPath string
}

type KafkaConfig struct {
	Brokers       []string
	ClientID      string // Optional but recommended
	DefaultTopic  string
	ConsumerGroup string
	EnableSSL     bool
}

type Mixcode struct {
	BaseURL       string
	ProgramID     string
	ProgramSecret string
}

type PubSubConfig struct {
	ProjectID             string
	SubscriptionID        string
	TopicID               string
	PubSubCredentialsPath string
}

type CDPConfig struct {
	BaseURL  string
	APIKey   string
	ClientID string
}

type CustomerDataServiceConfig struct {
	DatamartBaseURL    string
	HashedKOCIDBaseURL string
	APIKey             string
	ClientID           string
	Timeout            int // seconds
}

// getEnv is a helper function to read an environment variable or return a fallback value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// loadConfig loads the configuration from environment variables
func loadConfig() (*Config, error) {
	var config Config

	config.AppEnv = getEnv("APP_ENV", "development")
	config.AppPort = getEnv("PORT", "8080")
	config.APIKey = getEnv("API_KEY", "")
	config.XAPIKey = getEnv("X_API_KEY", "")

	config.InfobipConfig = InfobipConfig{
		BaseURL:     getEnv("INFOBIP_BASE_URL", ""),
		APIKey:      getEnv("INFOBIP_API_KEY", ""),
		MediaAPIKey: getEnv("INFOBIP_MEDIA_API_KEY", ""),
		WANumber:    getEnv("INFOBIP_WA_NUMBER", ""),
	}

	config.JwtConfig = JwtConfig{
		SecretKey:          getEnv("JWT_SECRET_KEY", ""),
		AccessTokenExpiry:  parseEnvInt("JWT_ACCESS_TOKEN_EXPIRY", 15),
		RefreshTokenExpiry: parseEnvInt("JWT_REFRESH_TOKEN_EXPIRY", 30),
		TempTokenExpiry:    parseEnvInt("JWT_TEMP_TOKEN_EXPIRY", 5),
	}

	config.SSHConfig = SSHConfig{
		UseSSH:      getEnv("USE_SSH", "false"),
		Host:        getEnv("SSH_HOST", ""),
		User:        getEnv("SSH_USER", ""),
		RDSEndpoint: getEnv("RDS_ENDPOINT", ""),
		RDSPort:     parseEnvInt("RDS_PORT", 5432),
		KafkaPort:   parseEnvInt("KAFKA_PORT", 9092),
	}

	endpoints := getEnv("KAFKA_ENDPOINT", "")
	var kafkaEndpoints []string
	if endpoints != "" {
		kafkaEndpoints = strings.Split(endpoints, ",")
	}
	config.SSHConfig.KafkaEndpoint = kafkaEndpoints

	config.DbConfig = DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     parseEnvInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "postgres"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}

	config.GluedinConfig = GluedinConfig{
		BaseURL:     getEnv("GLUEDIN_BASE_URL", ""),
		APIKey:      getEnv("GLUEDIN_API_KEY", ""),
		ProjectID:   getEnv("GLUEDIN_PROJECT_ID", ""),
		UGCEndpoint: getEnv("GLUEDIN_UGC_ENDPOINT", ""),
	}

	/*config.S3Config = S3Config{
		BucketName: getEnv("AWS_S3_BUCKET", ""),
		Region:     getEnv("AWS_REGION", ""),
		AccessKey:  getEnv("AWS_ACCESS_KEY", ""),
		SecretKey:  getEnv("AWS_SECRET_KEY", ""),
		OldS3Url:   getEnv("OLD_S3_URL", ""),
	}*/

	config.GcsConfig = GcsConfig{
		BucketName: getEnv("GCP_BUCKET_NAME", ""),
		ProjectID:  getEnv("GCP_PROJECT_ID", ""),
		GcpUrl:     getEnv("GCP_URL", ""),
	}

	config.FirebaseConfig = FirebaseConfig{
		ServiceKeyPath: getEnv("FIREBASE_SERVICE_KEY_PATH", ""),
	}

	kafkaBrokersEnv := getEnv("KAFKA_BROKERS", "")
	var kafkaBrokers []string
	if kafkaBrokersEnv != "" {
		kafkaBrokers = strings.Split(kafkaBrokersEnv, ",")
	}

	config.KafkaConfig = KafkaConfig{
		Brokers:       kafkaBrokers,
		ClientID:      getEnv("KAFKA_CLIENT_ID", "coca-cola-backend"),
		DefaultTopic:  getEnv("KAFKA_DEFAULT_TOPIC", ""),
		ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", ""),
		EnableSSL:     getEnv("KAFKA_ENABLE_SSL", "true") == "true",
	}

	config.Mixcode = Mixcode{
		BaseURL:       getEnv("MIXCODE_BASE_URL", ""),
		ProgramID:     getEnv("MIXCODE_PROGRAM_ID", ""),
		ProgramSecret: getEnv("MIXCODE_PROGRAM_SECRET", ""),
	}

	config.PubSubConfig = PubSubConfig{
		ProjectID:             getEnv("GOOGLE_PUBSUB_PROJECT_ID", ""),
		SubscriptionID:        getEnv("GOOGLE_PUBSUB_SUBSCRIPTION_ID", ""),
		TopicID:               getEnv("GOOGLE_PUBSUB_TOPIC_ID", ""),
		PubSubCredentialsPath: getEnv("GOOGLE_PUBSUB_CREDENTIALS_PATH", ""),
	}

	config.CDPConfig = CDPConfig{
		BaseURL:  getEnv("CDP_BASE_URL", ""),
		APIKey:   getEnv("CDP_API_KEY", ""),
		ClientID: getEnv("CDP_CLIENT_ID", ""),
	}

	config.CustomerDataServiceConfig = CustomerDataServiceConfig{
		DatamartBaseURL:    getEnv("CDS_DATAMART_BASE_URL", ""),
		HashedKOCIDBaseURL: getEnv("CDS_HASHED_KOCID_BASE_URL", ""),
		APIKey:             getEnv("CDS_API_KEY", ""),
		ClientID:           getEnv("CDS_CLIENT_ID", ""),
		Timeout:            parseEnvInt("CDS_TIMEOUT", 30),
	}

	config.ChatbotGluedinUserID = getEnv("CHATBOT_GLUEDIN_USER_ID", "")
	config.FrontendBaseURL = getEnv("FRONTEND_BASE_URL", "")

	config.MemeGeneratorConfig = MemeGeneratorConfig{
		FontPath:         getEnv("MEME_FONT_PATH", ""),
		BackdropURL:      getEnv("MEME_BACKDROP_URL", ""),
		LogoURL:          getEnv("MEME_LOGO_URL", ""),
		PythonServiceURL: getEnv("PY_IMAGE_SERVICE_URL", ""),
	}

	return &config, nil
}

// parseEnvInt is a helper function to parse an environment variable into an integer
func parseEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Printf("Failed to parse %s, using fallback %d", key, fallback)
	}
	return fallback
}

// GetConfig returns the singleton configuration instance
func GetConfig() *Config {
	once.Do(func() {
		var err error
		conf, err = loadConfig()
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	})
	return conf
}
