package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/desync-labs/tx-manager/executor/internal/config"
	messageBroker "github.com/desync-labs/tx-manager/executor/internal/message-broker"
	services "github.com/desync-labs/tx-manager/executor/internal/services"
	pb "github.com/desync-labs/tx-manager/executor/protos/key-manager"
	"google.golang.org/grpc"
)

// var appEnv = os.Getenv("APP_ENV")
// var grpcPortEnv = os.Getenv("GRPC_PORT_ENV")
// var rabbitMQUrl = os.Getenv("RABBITMQ_URL")

func main() {
	// Setting up default logger
	//Todo: Add log level as a configuration

	config, err := config.NewConfig()
	if err != nil {
		slog.Error("Error loading config: %v", err)
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

	// Initialize Redis client
	// redisClient := redis.NewClient(&redis.Options{
	// 	Addr: config.RedisUrl, // Redis server address
	// 	DB:   0,               // Default DB
	// })

	// _, err = redisClient.Ping().Result()
	// if err != nil {
	// 	slog.Error("Failed to connect to Redis: %v", err)
	// 	return
	// }

	var grpcOpts []grpc.DialOption
	grpcOpts = append(grpcOpts, grpc.WithInsecure())
	conn, err := grpc.NewClient(config.KeyManagerServerUrl, grpcOpts...)
	if err != nil {
		slog.Error("Failed to create gRPC connection: %v", err)
		return
	}
	keyManagerGrpcClient := pb.NewKeyManagerServiceClient(conn)

	defer conn.Close()

	messageBroker, err := messageBroker.NewRabbitMQ(config.RabitMQUrl, context.Background())
	if err != nil {
		slog.Error("Failed to create message broker: %v", err)
		return
	}

	priorities := []int{1, 2, 3}

	ctxExecutorService, cancelExecutorService := context.WithCancel(context.Background())
	defer cancelExecutorService()

	executorService := services.NewExecutorService(messageBroker, keyManagerGrpcClient, priorities, ctxExecutorService, 10)

	//Netwotkid and url to be moved to config
	evm := services.NewEVMTransaction("https://erpc.apothem.network")
	executorService.RegisterTransactionExecutor(51, evm)
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
