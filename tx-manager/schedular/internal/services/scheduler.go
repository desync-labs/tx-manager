package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/desync-labs/tx-manager/schedular/internal/domain"
	broker "github.com/desync-labs/tx-manager/schedular/internal/message-broker/interface"
)

// TODO: naming convention for interface
type SchedulerServiceInterface interface {
	ScheduleTransaction()
	SetupTransactionListener() error
}

type SchedulerService struct {
	messageBroker broker.MessageBrokerInterface
	// Map of priority to channel
	chTransactions map[string]chan domain.Transaction
	priorities     []string
	ctx            context.Context
	mu             sync.RWMutex
	cancel         context.CancelFunc
}

func NewSchedulerService(messageBroker broker.MessageBrokerInterface, priorities []string, ctx context.Context) *SchedulerService {
	ctxSchedularService, cancel := context.WithCancel(ctx)
	mb := &SchedulerService{
		messageBroker:  messageBroker,
		chTransactions: make(map[string]chan domain.Transaction),
		priorities:     priorities,
		ctx:            ctxSchedularService,
		cancel:         cancel,
	}
	return mb
}

func (s *SchedulerService) ScheduleTransaction() {
	// Schedule the transaction
	slog.Info("Scheduling transaction")
}

func (s *SchedulerService) SetupTransactionListener() error {
	// Listen for new transactions
	slog.Info("Setting up transaction listener")

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.priorities {
		s.chTransactions[p] = make(chan domain.Transaction)
		go s.listenForNewTransactionForPriority(p, s.chTransactions[p])
	}

	return nil
}

// priority will be p0, p1 or p2
func (s *SchedulerService) listenForNewTransactionForPriority(priority string, chNewTransaction chan domain.Transaction) {
	// Listen for new transactions
	slog.Info("Listening for new transactions", "priority", priority)

	s.messageBroker.ListenForSubmitterMessages(priority, func(body []byte, ctx context.Context) {
		tx := &domain.Transaction{}
		err := json.Unmarshal(body, tx)
		if err != nil {
			slog.Error("Failed to unmarshal message", "error", err)
			return
		} else {
			chNewTransaction <- *tx
		}
	})

	for {
		select {
		case tx := <-chNewTransaction:
			go s.scheduleTransaction(&tx)
		case <-s.ctx.Done():
			slog.Info("Shutting down transaction listener", "priority", priority)
			s.closePriorityChannel(priority)
			return
		}
	}
}

func (s *SchedulerService) closePriorityChannel(priority string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, exists := s.chTransactions[priority]; exists {
		close(ch)
		delete(s.chTransactions, priority)
		slog.Info("Closed channel for priority", "priority", priority)
	}
}

func (s *SchedulerService) scheduleTransaction(tx *domain.Transaction) {
	// Receive the transaction
	slog.Info("Schedule new transaction", "id", tx.Id)
}

func (s *SchedulerService) Shutdown() {
	slog.Info("Closing Scheduler Service")
	s.cancel()

	s.mu.Lock()
	for p, ch := range s.chTransactions {
		close(ch)
		delete(s.chTransactions, p)
		slog.Info("Closed channel during shutdown", "priority", p)
	}
	s.mu.Unlock()

	slog.Info("Scheduler Service shut down gracefully")
}
