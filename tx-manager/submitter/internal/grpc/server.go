package grpc

import (
	"log/slog"
	"net"

	pb "github.com/desync-labs/tx-manager/submitter/protos/transaction"
	"google.golang.org/grpc"
)

type TransactionSubmitterAdapter struct {
	//api ports.TransactionSubmitterPort // TransactionSubmitter logic
}

// NewAdapter creates a new Adapter
func NewTransactionSubmitterAdapter() *TransactionSubmitterAdapter {
	return &TransactionSubmitterAdapter{} //api: api}
}

func (adapter *TransactionSubmitterAdapter) StartGRPCServer(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	transactionSubmitterServer := adapter
	grpcServer := grpc.NewServer()
	pb.RegisterTransactionSubmitterServer(grpcServer, transactionSubmitterServer)

	slog.Info("gRPC server listening on port :" + port)
	return grpcServer.Serve(lis)
}
