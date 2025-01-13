package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/desync-labs/tx-manager/submitter/protos/transaction"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type TransactionSubmitterAdapter struct {
	//api ports.TransactionSubmitterPort // TransactionSubmitter logic
}

// NewAdapter creates a new Adapter
func NewTransactionSubmitterAdapter() *TransactionSubmitterAdapter {
	return &TransactionSubmitterAdapter{} //api: api}
}

// StartGRPCServer starts the gRPC server in a goroutine and handles graceful shutdown
func (adapter *TransactionSubmitterAdapter) StartGRPCServer(port string) error {
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

func (s *TransactionSubmitterAdapter) SubmitTransaction(ctx context.Context, req *pb.TransactionRequest) (*pb.TransactionResponse, error) {
	// tx := application.Transaction{
	// 	AppName:  req.GetAppName(),
	// 	Priority: req.GetPriority(),
	// 	Data:     req.GetTxData(),
	// }

	// // Use the transaction submitter to process the transaction
	// // txKey, err := s.submitter.SubmitTransaction(tx)
	// if err != nil {
	// 	log.Printf("Failed to submit transaction: %v", err)
	// 	return nil, err
	// }

	slog.Info("Received transaction request: %v", req)

	return &pb.TransactionResponse{
		TxKey:  "1",
		Status: pb.TransactionStatus_SUBMITTING,
	}, nil
}
