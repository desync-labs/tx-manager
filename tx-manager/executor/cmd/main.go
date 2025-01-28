package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	cache "github.com/desync-labs/tx-manager/executor/internal/cache"
	"github.com/desync-labs/tx-manager/executor/internal/config"
	messageBroker "github.com/desync-labs/tx-manager/executor/internal/message-broker"
	services "github.com/desync-labs/tx-manager/executor/internal/services"
	pb "github.com/desync-labs/tx-manager/executor/protos/key-manager"
	"github.com/hashicorp/vault/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	config, err := config.NewConfig()
	if err != nil {
		slog.Error("Error loading config", "error", err)
		panic(err)
	}

	logOpts := slog.LevelDebug
	if config.Env == "production" {
		logOpts = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logOpts,
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	slog.Info("Starting service...", "service name", config.GetApplicationName())

	var grpcOpts []grpc.DialOption
	grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.NewClient(config.KeyManagerServerUrl, grpcOpts...)
	if err != nil {
		slog.Error("Failed to create gRPC connection: %v", err)
		return
	}
	keyManagerGrpcClient := pb.NewKeyManagerServiceClient(conn)

	defer conn.Close()

	messageBroker, err := messageBroker.NewRabbitMQ(config.RabitMQUrl, context.Background())
	if err != nil {
		slog.Error("Failed to create message broker", "error", err)
		return
	}

	priorities := []int{1, 2, 3}

	ctxExecutorService, cancelExecutorService := context.WithCancel(context.Background())
	defer cancelExecutorService()

	// Initialize Vault client
	vaultConfig := api.DefaultConfig()
	vaultConfig.Address = config.VaultAddress

	slog.Debug("VaultAddress", "addr", vaultConfig.Address)

	if vaultConfig.Address == "" {
		log.Fatal("VAULT_ADDR environment variable not set")
	}

	client, err := api.NewClient(vaultConfig)
	if err != nil {
		log.Fatalf("Failed to create Vault client: %v", err)
	}

	// Set Vault token
	vaultToken := config.VaultToken

	slog.Debug("vaultToken", "token", vaultToken)

	if vaultToken == "" {
		log.Fatal("VAULT_TOKEN environment variable not set")
	}
	client.SetToken(vaultToken)

	// Initialize cache with TTL of 5 minutes
	cacheTTL := 5 * time.Minute

	cache := cache.NewKeyCache(client, cacheTTL)
	executorService := services.NewExecutorService(messageBroker, keyManagerGrpcClient, cache, priorities, ctxExecutorService, 10)

	factory := services.NewExecutorFactory()
	for _, chain := range config.ChainConfigs.Chains {
		executor, err := factory.CreateExecutor(chain.Type, chain.RPCURL)
		if err != nil {
			log.Printf("Error creating executor for chain %s: %v", chain.Name, err)
			continue
		}

		executorService.RegisterTransactionExecutor(chain.ID, executor)
		fmt.Printf("Registered executor for chain %s (ID: %d)\n", chain.Name, chain.ID)
	}

	executorService.SetupTransactionListener()

	// Listen for interrupt or termination signals for graceful shutdown
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-stopCh
	slog.Info("Received signal: " + sig.String() + ". Shutting down...")

	messageBroker.Close()

	// Graceful shutdown logic (optional additional cleanup)
	time.Sleep(2 * time.Second)
	slog.Info("Shutdown complete.")
}
