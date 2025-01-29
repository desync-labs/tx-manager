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
			slog.Error("gRPC server failed to start", "error", err)
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

	err := s.keyManagerService.AssignKey(req.TxId, int(req.GetPriority()), req.NetworkId, context.Background())

	if err != nil {
		slog.Error("Failed to assign key", "tx-id", req.TxId, "error", err)
		return &pb.KeyAssigmentResponse{
			Success: false,
		}, nil

	}

	return &pb.KeyAssigmentResponse{
		Success: true,
	}, nil
}

func (s *GrpcServer) GetKey(ctx context.Context, req *pb.KeyManagerRequest) (*pb.KeyFetchResponse, error) {

	slog.Debug("Received key fetch request for transaction", "tx-id", req.TxId)

	pk, err := s.keyManagerService.GetKey(req.TxId, int(req.GetPriority()), req.NetworkId, context.Background())

	if err != nil {
		slog.Error("Failed to fetch key", "tx-id", req.TxId, "error", err)
		return &pb.KeyFetchResponse{
			Key: "",
		}, err
	}

	return &pb.KeyFetchResponse{
		Key: string(pk),
	}, nil
}

func (s *GrpcServer) ReleaseKey(ctx context.Context, in *pb.KeyManagerRequest) (*pb.KeyReleaseResponse, error) {
	slog.Debug("Received key release request for transaction", "tx-id", in.TxId)

	err := s.keyManagerService.ReleaseKey(in.TxId, int(in.GetPriority()), in.NetworkId, context.Background())
	if err != nil {
		slog.Error("Failed to release key", "tx-id", in.TxId, "error", err)
		return &pb.KeyReleaseResponse{
			Success: false,
		}, nil
	}

	return &pb.KeyReleaseResponse{
		Success: true,
	}, nil
}
