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
	// Map of priority to channel
	chPriority map[string]chan domain.Transaction
	priorties  []string
	ctx        context.Context
}

func NewSchedularService(messageBroker broker.MessageBrokerInterface, priorties []string, ctx context.Context) *SchedularService {
	mb := &SchedularService{
		messageBroker: messageBroker,
		chPriority:    make(map[string]chan domain.Transaction),
		priorties:     priorties,
		ctx:           ctx,
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

	for _, p := range s.priorties {
		s.chPriority[p] = make(chan domain.Transaction)
		go s.listenForNewTransactionForPriority(p, s.chPriority[p])
	}

	return nil
}

// priority will be p0, p1 or p2
func (s *SchedularService) listenForNewTransactionForPriority(priority string, chNewTransaction chan domain.Transaction) {
	// Listen for new transactions
	slog.Info("Listening for new transactions", "priority", priority)

	//Why is this in a go routine?
	go func() {
		s.messageBroker.ListenForSubmitterMessages(priority, func(body []byte, ctx context.Context) {
			tx := &domain.Transaction{}
			err := json.Unmarshal(body, tx)
			if err != nil {
				slog.Error("Failed to unmarshal message", "error", err)
				// return
			} else {
				chNewTransaction <- *tx
			}
		})
	}()

	for {
		select {
		case tx := <-chNewTransaction:
			go s.recieveTransaction(&tx)
		case <-s.ctx.Done():
			slog.Info("Shutting down transaction listener", "priority", priority)
			s.close()
			return
		}
	}
}

func (s *SchedularService) recieveTransaction(tx *domain.Transaction) {
	// Recieve the transaction
	slog.Info("Recieved new transaction", "id", tx.Id)
}

func (s *SchedularService) close() {
	// Recieve the transaction
	slog.Info("Closing Service", "serivce", "schedular")
	for _, p := range s.priorties {
		close(s.chPriority[p])
		delete(s.chPriority, p)
	}
}
