package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	gRPC "github.com/desync-labs/tx-manager/submitter/internal/grpc"
)

var appEnv = os.Getenv("APP_ENV")
var grpcPortEnv = os.Getenv("GRPC_PORT_ENV")

func main() {
	// Setting up default logger
	//Todo: Add log level as a configuration

	logOpts := slog.LevelDebug
	if appEnv == "production" {
		logOpts = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logOpts,
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	slog.Info("Starting tx submitter service...")

	//TODO: Move to a config file
	if grpcPortEnv == "" {
		grpcPortEnv = "50051"
	}

	gRPCAdapter := gRPC.NewTransactionSubmitterAdapter()

	// Start gRPC server asynchronously in a goroutine
	go func() {
		if err := gRPCAdapter.StartGRPCServer(grpcPortEnv); err != nil {
			slog.Error("Failed to start gRPC server: %v", err)
		}
	}()

	// Listen for interrupt or termination signals for graceful shutdown
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-stopCh
	slog.Info("Received signal: " + sig.String() + ". Shutting down...")

	// Graceful shutdown logic (optional additional cleanup)
	time.Sleep(2 * time.Second)
	slog.Info("Shutdown complete.")
}
