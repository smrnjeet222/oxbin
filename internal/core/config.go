package core

import "time"

// Config holds all application configuration
type Config struct {
	Walrus WalrusConfig `json:"walrus"`
	CLI    CLIConfig    `json:"cli"`
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
	return &Config{
		Walrus: WalrusConfig{
			PublisherURLs: []string{
				"https://publisher.walrus-testnet.walrus.space",
				"https://walrus-publisher-testnet.staking4all.org",
			},
			AggregatorURLs: []string{
				"https://aggregator.walrus-testnet.walrus.space",
				"https://walrus-testnet-aggregator.staking4all.org",
			},
			HTTPTimeout:   30 * time.Second,
			UploadTimeout: 60 * time.Second,
			MaxFileSize:   10 * 1024 * 1024, // 10MB
		},
		CLI: CLIConfig{
			DefaultWidth:  80,
			DefaultHeight: 24,
			MinWidth:      40,
			MinHeight:     10,
		},
	}
}

// App constants
const (
	AppName    = "OxBin"
	AppVersion = "1.0.0"
)
