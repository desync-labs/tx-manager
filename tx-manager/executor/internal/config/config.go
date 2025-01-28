package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

const Environment = "GO_ENV"
const RabbitMQUrlKey = "RABBITMQ_URL"
const RabbitMQUsernameKey = "RABBITMQ_USERNAME"
const RabbitMQUrlPasswordKey = "RABBITMQ_PASSWORD"
const KeyManagerServerKey = "GRPC_KEY_MANAGER_URL"

const VaultAddresskey = "VAULT_ADDR"
const VaultTokenKey = "VAULT_TOKEN"

//const RedisUrlKey = "REDIS_URL"

// ChainConfig represents the configuration for a single blockchain network.
type ChainConfig struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"` // e.g., "evm", "solana"
	RPCURL string `json:"rpc_url"`
}

// AppConfig holds the entire application configuration.
type ChainConfigs struct {
	Chains []ChainConfig `json:"chains"`
}

type Config struct {
	// PortNumber string
	Env                 string
	RabitMQUrl          string
	KeyManagerServerUrl string
	VaultAddress        string
	VaultToken          string

	ChainConfigs *ChainConfigs
}

func NewConfig() (*Config, error) {

	chainConfigs, err := loadChainConfig("./internal/config/chains_config.json")
	if err != nil {
		slog.Error("Error loading chain config", "error", err)
		return nil, err
	}

	if _, err := os.Stat(".env"); err == nil {
		// Load environment variables from the .env file
		if err := godotenv.Load(); err != nil {
			slog.Error("Error loading .env file", "error", err)
			return nil, err
		}
	}

	env := os.Getenv(Environment)

	rabitMQUsername := os.Getenv(RabbitMQUsernameKey)
	rabitMQPassword := os.Getenv(RabbitMQUrlPasswordKey)
	rabitMQUrl := os.Getenv(RabbitMQUrlKey)
	KeyManagerServerUrl := os.Getenv(KeyManagerServerKey)

	VaultAddress := os.Getenv(VaultAddresskey)
	VaultToken := os.Getenv(VaultTokenKey)

	//Error loading environment variables
	if env == "" ||
		rabitMQUsername == "" ||
		rabitMQPassword == "" ||
		rabitMQUrl == "" ||
		KeyManagerServerUrl == "" ||
		VaultAddress == "" ||
		VaultToken == "" {
		slog.Error("Error loading data from environment")
		return nil, fmt.Errorf("error loading data from environment")
	}

	rabitMQUrl = "amqp://" + rabitMQUsername + ":" + rabitMQPassword + "@" + rabitMQUrl

	return &Config{
		Env: env, RabitMQUrl: rabitMQUrl, KeyManagerServerUrl: KeyManagerServerUrl, VaultAddress: VaultAddress, VaultToken: VaultToken, ChainConfigs: chainConfigs}, nil
}

// LoadConfig reads the configuration from the specified JSON file.
func loadChainConfig(filePath string) (*ChainConfigs, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config ChainConfigs
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %v", err)
	}

	return &config, nil
}

func (s *Config) GetEnvironment() string {
	return s.Env
}

func (s *Config) GetApplicationName() string {
	return "tx-executor"
}
