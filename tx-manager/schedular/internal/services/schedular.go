package services

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/desync-labs/tx-manager/schedular/internal/domain"
	broker "github.com/desync-labs/tx-manager/schedular/internal/message-broker/interface"
)

type SchedularServiceInterface interface {
	// ScheduleTransaction schedules the transaction
	ScheduleTransaction() (string, error)
	//Recieve the transaction from message broker
	RecieveTransaction()
	// SetupTransactionListner sets up the transaction listener
	SetupTransactionListner() error
}

type SchedularService struct {
	messageBroker broker.MessageBrokerInterface
}

func NewSchedularService(messageBroker broker.MessageBrokerInterface) *SchedularService {
	mb := &SchedularService{
		messageBroker: messageBroker,
	}
	return mb
}

func (s *SchedularService) ScheduleTransaction() {
	// Schedule the transaction
	slog.Info("Scheduling transaction")
}

func (s *SchedularService) SetupTransactionListner() error {
	// Listen for new transactions
	slog.Info("Setting up transaction listener")

	//	go s.listenForNewTransactionForPriority("p0")

	// Listen for new transactions for priority p0
	err := s.listenForNewTransactionForPriority("p1")
	if err != nil {
		slog.Error("Failed to listen for new transactions", "error", err)
		return err
	}

	// Listen for new transactions for priority p1
	err = s.listenForNewTransactionForPriority("p2")
	if err != nil {
		slog.Error("Failed to listen for new transactions", "error", err)
		return err
	}

	// Listen for new transactions for priority p2
	err = s.listenForNewTransactionForPriority("p3")
	if err != nil {
		slog.Error("Failed to listen for new transactions", "error", err)
		return err
	}

	return nil
}

// priority will be p0, p1 or p2
func (s *SchedularService) listenForNewTransactionForPriority(priority string) error {
	// Listen for new transactions
	slog.Info("Listening for new transactions", "priority", priority)

	err := s.messageBroker.ListenForSubmitterMessages(priority, func(body []byte, ctx context.Context) {
		tx := &domain.Transaction{}
		err := json.Unmarshal(body, tx)
		if err != nil {
			slog.Error("Failed to unmarshal message", "error", err)
			return
		}

		s.recieveTransaction(tx)
	})

	if err != nil {
		slog.Error("Failed to listen for messages", "error", err)
		return err
	}

	return nil
}

func (s *SchedularService) recieveTransaction(tx *domain.Transaction) {
	// Recieve the transaction
	slog.Info("Recieved new transaction", "id", tx.Id)
}
