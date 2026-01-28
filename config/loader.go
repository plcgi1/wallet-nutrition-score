package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Port       int    `yaml:"port"`
		LogLevel   string `yaml:"log_level"`
		TimeoutSec int    `yaml:"timeout_sec"`
	} `yaml:"app"`
	GoPlus struct {
		ApiKey    string `yaml:"key"`
		ApiSecret string `yaml:"secret"`
	}
	Etherscan struct {
		URL    string `yaml:"url"`
		ApiKey string `yaml:"key"`
	}
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	Scoring struct {
		BaseScore float64            `yaml:"base_score"`
		Weights   map[string]float64 `yaml:"weights"`
	} `yaml:"scoring"`
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func Load() (*Config, error) {
	// Load environment variables from .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	// Read YAML config file
	configPath := filepath.Join("config", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse YAML
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// Override with environment variables if provided
	if goplusKey := getEnv("GOPLUS_API_KEY", ""); goplusKey != "" {
		config.GoPlus.ApiKey = goplusKey
	}
	if goplusSecret := getEnv("GOPLUS_API_SECRET", ""); goplusSecret != "" {
		config.GoPlus.ApiSecret = goplusSecret
	}
	if etherscanURL := getEnv("ETHERSCAN_URL", ""); etherscanURL != "" {
		config.Etherscan.URL = etherscanURL
	}
	if etherscanApiKey := getEnv("ETHERSCAN_API_KEY", ""); etherscanApiKey != "" {
		config.Etherscan.ApiKey = etherscanApiKey
	}
	if redisAddr := getEnv("REDIS_ADDR", ""); redisAddr != "" {
		config.Redis.Addr = redisAddr
	}
	if redisPassword := getEnv("REDIS_PASSWORD", ""); redisPassword != "" {
		config.Redis.Password = redisPassword
	}
	if redisDB := getEnv("REDIS_DB", ""); redisDB != "" {
		var db int
		_, err = fmt.Sscanf(redisDB, "%d", &db)
		if err == nil {
			config.Redis.DB = db
		}
	}
	return &config, nil
}
