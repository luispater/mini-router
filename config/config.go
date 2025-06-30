package config

import (
	"fmt"
	"github.com/luispater/mini-router/models"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application's configuration
type Config struct {
	Server  ServerConfig    `yaml:"server"`
	Models  []models.Model  `yaml:"models"`
	APIKeys []models.APIKey `yaml:"api_keys"`
}

// ServerConfig represents the server's configuration
type ServerConfig struct {
	// Port to listen on
	Port string `yaml:"port"`
	// ShutdownTimeout is the timeout for graceful shutdown
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// / LoadConfig loads the configuration from the specified file
func LoadConfig(configFile string) (*Config, error) {
	// Read the configuration file
	data, err := os.ReadFile(configFile)
	// If reading the file fails
	if err != nil {
		// Return an error
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the YAML data
	var config Config
	// If parsing the YAML data fails
	if err = yaml.Unmarshal(data, &config); err != nil {
		// Return an error
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Return the configuration
	return &config, nil
}
