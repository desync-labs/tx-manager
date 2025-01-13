package main

import (
	"log/slog"
	"os"

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

	err := gRPCAdapter.StartGRPCServer(grpcPortEnv)
	if err != nil {
		slog.Error("Failed to start gRPC server: %v", err)
		return
	}

	slog.Info("Started tx submitter service...")
}
