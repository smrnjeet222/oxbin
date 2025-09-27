package core

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Walrus    WalrusConfig `json:"walrus"`
	CLI       CLIConfig    `json:"cli"`
	WebAppURL string       `json:"web_app_url"`
}

// WalrusConfig contains Walrus-specific configuration
type WalrusConfig struct {
	PublisherURLs  []string      `json:"publisher_urls"`
	AggregatorURLs []string      `json:"aggregator_urls"`
	HTTPTimeout    time.Duration `json:"http_timeout"`
	UploadTimeout  time.Duration `json:"upload_timeout"`
	MaxFileSize    int64         `json:"max_file_size"`
}

// CLIConfig contains CLI-specific configuration
type CLIConfig struct {
	DefaultWidth  int `json:"default_width"`
	DefaultHeight int `json:"default_height"`
	MinWidth      int `json:"min_width"`
	MinHeight     int `json:"min_height"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	// Load .env file if it exists (ignore errors for optional .env files)
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using environment variables and defaults: %v", err)
	}

	// Get environment variables with defaults
	webAppUrl := getEnvOrDefault("OXBIN_WEBAPP_URL", "http://localhost:8080/blob/")

	// Parse publisher URLs
	publisherURLs := []string{
		"https://publisher.walrus-testnet.walrus.space",
		"https://walrus-publisher-testnet.staking4all.org",
	}

	// Parse aggregator URLs
	aggregatorURLs := []string{
		"https://aggregator.walrus-testnet.walrus.space",
	}

	return &Config{
		Walrus: WalrusConfig{
			PublisherURLs:  publisherURLs,
			AggregatorURLs: aggregatorURLs,
			HTTPTimeout:    30 * time.Second,
			UploadTimeout:  60 * time.Second,
			MaxFileSize:    10 * 1024 * 1024, // 10MB
		},
		CLI: CLIConfig{
			DefaultWidth:  80,
			DefaultHeight: 24,
			MinWidth:      40,
			MinHeight:     10,
		},
		WebAppURL: webAppUrl,
	}
}

// getEnvOrDefault returns the environment variable value or a default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// App constants
const (
	AppName    = "OxBin"
	AppVersion = "1.0.0"
)
