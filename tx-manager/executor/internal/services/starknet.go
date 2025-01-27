package services

import (
	"log/slog"

	"github.com/desync-labs/tx-manager/executor/internal/domain"
)

type StarknetTransaction struct {
	url string
}

func NewStarknetTransaction(url string) *StarknetTransaction {
	return &StarknetTransaction{
		url: url,
	}
}

func (s *StarknetTransaction) Execute(key string, tx *domain.Transaction) error {
	slog.Error("Service not implemented", "service", "starknet", "id", tx.Id)
	return nil
}
