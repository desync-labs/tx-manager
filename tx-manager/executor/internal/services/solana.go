package services

import (
	"log/slog"

	"github.com/desync-labs/tx-manager/executor/internal/domain"
)

var _ TransactionExecutorInterface = (*SolanaTransaction)(nil)

type SolanaTransaction struct {
	url string
}

func NewSolanaTransaction(url string) *SolanaTransaction {
	return &SolanaTransaction{
		url: url,
	}
}

func (s *SolanaTransaction) Execute(key string, tx *domain.Transaction) (bool, string, error) {
	slog.Error("Service not implemented", "service", "solana", "id", tx.Id)
	return false, "", nil
}
