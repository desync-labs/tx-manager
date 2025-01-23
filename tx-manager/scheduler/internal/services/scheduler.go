package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/desync-labs/tx-manager/scheduler/internal/domain"
	broker "github.com/desync-labs/tx-manager/scheduler/internal/message-broker/interface"
)

const (
	// Topic to submit transactions, for rabbitmq this is exchange name
	submit_topic = "tx_submit"
)

// TODO: naming convention for interface
type SchedulerServiceInterface interface {
	ScheduleTransaction()
	SetupTransactionListener() error
}

type SchedulerService struct {
	messageBroker broker.MessageBrokerInterface
	// Map of priority to channel
	chTransactions map[int]chan domain.Transaction
	priorities     []int
	ctx            context.Context
	mu             sync.RWMutex
	cancel         context.CancelFunc
	workerPoolSize int
	taskQueue      chan domain.Transaction
	wg             sync.WaitGroup
}

func NewSchedulerService(messageBroker broker.MessageBrokerInterface, priorities []int, ctx context.Context, workerPoolSize int) *SchedulerService {
	ctxSchedularService, cancel := context.WithCancel(ctx)
	mb := &SchedulerService{
		messageBroker:  messageBroker,
		chTransactions: make(map[int]chan domain.Transaction),
		priorities:     priorities,
		ctx:            ctxSchedularService,
		cancel:         cancel,
		workerPoolSize: workerPoolSize,
		taskQueue:      make(chan domain.Transaction, 100),
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

	// Start worker pool to process the transactions as they recieve from message broker
	for i := 0; i < s.workerPoolSize; i++ {
		s.wg.Add(1)
		go s.worker(i + 1)
	}

	for _, p := range s.priorities {
		s.chTransactions[p] = make(chan domain.Transaction)
		go s.listenForNewTransactionForPriority(p, s.chTransactions[p])
	}

	return nil
}

// priority will be p0, p1 or p2
func (s *SchedulerService) listenForNewTransactionForPriority(priority int, chNewTransaction chan domain.Transaction) {
	// Listen for new transactions
	slog.Info("Listening for new transactions", "priority", priority)

	s.messageBroker.ListenForMessages(submit_topic, priority, func(body []byte, ctx context.Context) {
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
		case tx, ok := <-chNewTransaction:
			if !ok {
				slog.Info("Transaction channel closed", "priority", priority)
				return
			}
			select {
			case s.taskQueue <- tx:
			case <-s.ctx.Done():
				return
			}
		case <-s.ctx.Done():
			slog.Info("Shutting down transaction listener", "priority", priority)
			s.closePriorityChannel(priority)
			return
		}
	}
}

func (s *SchedulerService) closePriorityChannel(priority int) {
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

// Worker function to process tasks
func (s *SchedulerService) worker(id int) {
	defer s.wg.Done()
	slog.Debug("Worker started", "worker_id", id)
	for {
		select {
		case tx, ok := <-s.taskQueue:
			if !ok {
				slog.Debug("Worker stopping", "worker_id", id)
				return
			}
			s.scheduleTransaction(&tx)
		case <-s.ctx.Done():
			slog.Debug("Worker received shutdown signal", "worker_id", id)
			return
		}
	}
}
