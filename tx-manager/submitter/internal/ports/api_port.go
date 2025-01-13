package ports

import (
	"context"

	pb "github.com/desync-labs/tx-manager/submitter/protos/transaction"
)

// APIPort is the technology neutral
// port for driving adapters
type TransactionSubmitterPort interface {
	SubmitTransaction(context.Context, *pb.TransactionRequest) (*pb.TransactionResponse, error)
}
