package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/desync-labs/tx-manager/submitter/internal/config"
	gRPC "github.com/desync-labs/tx-manager/submitter/internal/grpc"
	messageBroker "github.com/desync-labs/tx-manager/submitter/internal/message-broker"
	services "github.com/desync-labs/tx-manager/submitter/internal/service"
	"github.com/go-redis/redis"
)

// var appEnv = os.Getenv("APP_ENV")
// var grpcPortEnv = os.Getenv("GRPC_PORT_ENV")
// var rabbitMQUrl = os.Getenv("RABBITMQ_URL")

func main() {
	// Setting up default logger
	//Todo: Add log level as a configuration

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

	slog.Info("Starting tx submitter service...")

	slog.Debug("Redis URL", "url", config.RedisUrl)

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisUrl, // Redis server address
		DB:   0,               // Default DB
	})

	_, err = redisClient.Ping().Result()
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		return
	}

	//TODO: Move to a config file
	grpcPortEnv := config.PortNumber

	messageBroker, err := messageBroker.NewRabbitMQ(config.RabitMQUrl, context.Background())
	if err != nil {
		slog.Error("Failed to create message broker", "error", err)
		return
	}

	submitterService := services.NewSubmitterService(messageBroker, redisClient)
	//submitterService.SetupTransactionStatusEvent()

	grpcServer := gRPC.NewGrpcServer(submitterService)

	// Start gRPC server asynchronously in a goroutine
	go func() {
		if err := grpcServer.Start(grpcPortEnv); err != nil {
			slog.Error("Failed to start gRPC server", "error", err)
		}
	}()

	// Listen for interrupt or termination signals for graceful shutdown
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-stopCh
	slog.Info("Received signal: " + sig.String() + ". Shutting down...")

	redisClient.Close()
	messageBroker.Close()

	// Graceful shutdown logic (optional additional cleanup)
	time.Sleep(2 * time.Second)
	slog.Info("Shutdown complete.")
}
