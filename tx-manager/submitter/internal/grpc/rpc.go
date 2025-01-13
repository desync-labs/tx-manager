package grpc

import (
	"context"

	pb "github.com/desync-labs/tx-manager/submitter/protos/transaction"
)

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

	return &pb.TransactionResponse{
		TxKey:  "1",
		Status: pb.TransactionStatus_SUBMITTING,
	}, nil
}
