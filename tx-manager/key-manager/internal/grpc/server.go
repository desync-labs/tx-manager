package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	services "github.com/desync-labs/tx-manager/key-manager/internal/services"
	pb "github.com/desync-labs/tx-manager/key-manager/protos/key-manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GrpcServer struct {
	keyManagerService *services.KeyManagerService // TransactionSubmitter logic
}

// NewAdapter creates a new Adapter
func NewGrpcServer(keyManagerService *services.KeyManagerService) *GrpcServer {
	return &GrpcServer{
		keyManagerService: keyManagerService,
	}
}

// StartGRPCServer starts the gRPC server in a goroutine and handles graceful shutdown
func (adapter *GrpcServer) Start(port string) error {
	// Create a TCP listener on the specified port
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	// Create a new gRPC server
	grpcServer := grpc.NewServer()

	//TODO: Add env variable to enable/disable reflection
	reflection.Register(grpcServer)

	// Register the TransactionSubmitter server with gRPC
	keyManagerServer := adapter
	pb.RegisterKeyManagerServiceServer(grpcServer, keyManagerServer)

	// Set up graceful shutdown handling
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	// Run gRPC server in a goroutine
	go func() {
		slog.Info("gRPC server listening on port: " + port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed to start: %v", err)
		}
	}()

	// Block until we receive a signal to stop (SIGINT or SIGTERM)
	sig := <-stopCh
	slog.Info(fmt.Sprintf("Received signal: %v. Shutting down...", sig))

	// Graceful shutdown logic
	grpcServer.GracefulStop()
	slog.Info("gRPC server stopped gracefully")
	return nil
}

func (s *GrpcServer) AssignKey(ctx context.Context, req *pb.KeyManagerRequest) (*pb.KeyAssigmentResponse, error) {

	slog.Debug("Received key assignment request for transaction", "tx-id", req.TxId)

	return &pb.KeyAssigmentResponse{
		Success: true,
	}, nil
}

func (s *GrpcServer) GetKey(context.Context, *pb.KeyManagerRequest) (*pb.KeyFetchResponse, error) {
	return &pb.KeyFetchResponse{
		Key: "test-key",
	}, nil
}
