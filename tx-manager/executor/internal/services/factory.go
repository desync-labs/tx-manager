// services/factory.go

package services

import (
	"fmt"
)

// ExecutorFactory creates TransactionExecutor instances based on chain type.
type ExecutorFactory struct{}

func NewExecutorFactory() *ExecutorFactory {
	return &ExecutorFactory{}
}

func (f *ExecutorFactory) CreateExecutor(chainType, rpcURL string) (TransactionExecutorInterface, error) {
	switch chainType {
	case "evm":
		return NewEVMTransaction(rpcURL), nil
	case "solana":
		return NewSolanaTransaction(rpcURL), nil
	// Add cases for other chain types
	default:
		return nil, fmt.Errorf("unsupported chain type: %s", chainType)
	}
}
