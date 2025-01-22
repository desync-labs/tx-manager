package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

const Environment = "GO_ENV"
const RabbitMQUrlKey = "RABBITMQ_URL"
const RabbitMQUsernameKey = "RABBITMQ_USERNAME"
const RabbitMQUrlPasswordKey = "RABBITMQ_PASSWORD"

//const RedisUrlKey = "REDIS_URL"

type Config struct {
	// PortNumber string
	Env        string
	RabitMQUrl string
	// RedisUrl   string
}

func NewConfig() (*Config, error) {

	if _, err := os.Stat(".env"); err == nil {
		// Load environment variables from the .env file
		if err := godotenv.Load(); err != nil {
			slog.Error("Error loading .env file:", err)
			return nil, err
		}
	}

	env := os.Getenv(Environment)

	rabitMQUsername := os.Getenv(RabbitMQUsernameKey)
	rabitMQPassword := os.Getenv(RabbitMQUrlPasswordKey)
	rabitMQUrl := os.Getenv(RabbitMQUrlKey)
	// redisUrl := os.Getenv(RedisUrlKey)

	//Error loading environment variables
	if env == "" ||
		rabitMQUsername == "" ||
		rabitMQPassword == "" ||
		rabitMQUrl == "" {
		//redisUrl == "" {
		slog.Error("Error loading data from environment")
		return nil, fmt.Errorf("Error loading data from environment")
	}

	rabitMQUrl = "amqp://" + rabitMQUsername + ":" + rabitMQPassword + "@" + rabitMQUrl

	return &Config{
		Env: env, RabitMQUrl: rabitMQUrl}, nil
}

func (s *Config) GetEnvironment() string {
	return s.Env
}

func (s *Config) GetApplicationName() string {
	return "tx-schedular"
}
