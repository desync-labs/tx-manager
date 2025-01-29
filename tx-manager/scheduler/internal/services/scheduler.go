package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/desync-labs/tx-manager/scheduler/internal/domain"
	mb "github.com/desync-labs/tx-manager/scheduler/internal/message-broker"
	broker "github.com/desync-labs/tx-manager/scheduler/internal/message-broker/interface"
	pb "github.com/desync-labs/tx-manager/scheduler/protos/key-manager"
)

type SchedulerServiceInterface interface {
	SetupTransactionListener() error
}

type SchedulerService struct {
	messageBroker broker.MessageBrokerInterface
	// Map of priority to channel
	chTransactions          map[int]chan domain.Transaction
	priorities              []int
	ctx                     context.Context
	mu                      sync.RWMutex
	cancel                  context.CancelFunc
	workerPoolSize          int
	taskQueue               chan domain.Transaction
	wg                      sync.WaitGroup
	keyManagerServiceClient pb.KeyManagerServiceClient
}

func NewSchedulerService(messageBroker broker.MessageBrokerInterface, keyManagerServiceClient pb.KeyManagerServiceClient, priorities []int, ctx context.Context, workerPoolSize int) *SchedulerService {
	ctxSchedularService, cancel := context.WithCancel(ctx)
	mb := &SchedulerService{
		messageBroker:           messageBroker,
		chTransactions:          make(map[int]chan domain.Transaction),
		priorities:              priorities,
		ctx:                     ctxSchedularService,
		cancel:                  cancel,
		workerPoolSize:          workerPoolSize,
		taskQueue:               make(chan domain.Transaction, 100),
		keyManagerServiceClient: keyManagerServiceClient,
	}
	return mb
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

	s.messageBroker.ListenForMessages(mb.Submit_Exchange, priority, func(body []byte, ctx context.Context) {
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

// Use 2-phase commit approach to first lock the key for transaction and then call executer to perform execution
func (s *SchedulerService) scheduleTransaction(tx *domain.Transaction) {
	// Receive the transaction
	slog.Info("Schedule new transaction", "id", tx.Id)
	resp, err := s.keyManagerServiceClient.AssignKey(context.Background(), &pb.KeyManagerRequest{
		TxId:     tx.Id,
		Priority: pb.Priority(tx.Priority),
	})

	//TODO: Handle error, either retry or add to dead letter queue
	if err != nil {
		slog.Error("Failed to assign key, add transaction to dead letter queue", "error", err)
		return
	}

	//TODO: Handle no key, add to dead letter queue
	if !resp.Success {
		slog.Error("No key avaiable, adding to dead letter queue", "tx-id", tx.Id)

		// Publish transaction status to message broker
		tx_status := &domain.TransactionStatus{
			Id:       tx.Id,
			Status:   domain.Tx_Status_Error,
			At:       time.Now(),
			Response: fmt.Sprintf("No key available for transaction %s", tx.Id),
		}
		s.messageBroker.PublishObject(mb.Tx_Status_Exchange, tx_status, -1, context.Background())
		return
	}

	slog.Info("Key assigned successfully", "tx-id", tx.Id)
	s.messageBroker.PublishObject(mb.Executor_Exchange, tx, tx.Priority, context.Background())

	// Publish transaction status to message broker
	tx_status := &domain.TransactionStatus{
		Id:       tx.Id,
		Status:   domain.Tx_Status_Scheduled,
		At:       time.Now(),
		Response: "Transaction scheduled for execution",
	}
	s.messageBroker.PublishObject(mb.Tx_Status_Exchange, tx_status, -1, context.Background())
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
