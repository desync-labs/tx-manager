package services

import (
	"log/slog"

	"github.com/desync-labs/tx-manager/executor/internal/domain"
)

type SolanaTransaction struct {
	url string
}

func NewSolanaTransaction(url string) *SolanaTransaction {
	return &SolanaTransaction{
		url: url,
	}
}

func (s *SolanaTransaction) Execute(key string, tx *domain.Transaction) error {
	slog.Error("Service not implemented", "service", "solana", "id", tx.Id)
	return nil
}
