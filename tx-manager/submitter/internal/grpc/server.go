package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	services "github.com/desync-labs/tx-manager/submitter/internal/service"
	pb "github.com/desync-labs/tx-manager/submitter/protos/transaction"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GrpcServer struct {
	submitterService services.SubmitterServiceInterface // TransactionSubmitter logic
}

// NewAdapter creates a new Adapter
func NewGrpcServer(submitterService services.SubmitterServiceInterface) *GrpcServer {
	return &GrpcServer{
		submitterService: submitterService,
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
	transactionSubmitterServer := adapter
	pb.RegisterTransactionSubmitterServer(grpcServer, transactionSubmitterServer)

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

func (s *GrpcServer) SubmitTransaction(ctx context.Context, req *pb.TransactionRequest) (*pb.TransactionResponse, error) {

	slog.Debug("Received transaction request: %v", req)

	txId, err := s.submitterService.SubmitTransaction(req)
	if err != nil {
		slog.Error("Failed to submit transaction: %v", err)
		return nil, err
	}

	return &pb.TransactionResponse{
		TxKey:  txId,
		Status: pb.TransactionStatus_SUBMITTING,
	}, nil
}
