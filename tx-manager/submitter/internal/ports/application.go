package ports

import (
	"context"

	pb "github.com/desync-labs/tx-manager/submitter/protos/transaction"
)

// port for driving adapters
type SubmitterPort interface {
	SubmitTransaction(context.Context, *pb.TransactionRequest) (*pb.TransactionResponse, error)
}
