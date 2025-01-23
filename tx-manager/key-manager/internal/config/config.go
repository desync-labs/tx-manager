package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

const Environment = "GO_ENV"
const LocalPortNumber = "GRPC_PORT_ENV"
const RedisUrlKey = "REDIS_URL"

type Config struct {
	PortNumber string
	Env        string
	RedisUrl   string
}

func NewConfig() (*Config, error) {

	if _, err := os.Stat(".env"); err == nil {
		// Load environment variables from the .env file
		if err := godotenv.Load(); err != nil {
			slog.Error("Error loading .env file:", err)
			return nil, err
		}
	}

	localPortNumber := os.Getenv(LocalPortNumber)
	env := os.Getenv(Environment)

	redisUrl := os.Getenv(RedisUrlKey)

	//Error loading environment variables
	if localPortNumber == "" ||
		env == "" ||
		redisUrl == "" {
		slog.Error("Error loading data from environment")
		return nil, fmt.Errorf("Error loading data from environment")
	}

	return &Config{
		PortNumber: localPortNumber, Env: env, RedisUrl: redisUrl}, nil
}

func (s *Config) GetEnvironment() string {
	return s.Env
}

func (s *Config) GetApplicationName() string {
	return "key-manager"
}
