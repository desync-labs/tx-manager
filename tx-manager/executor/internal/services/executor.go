package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/desync-labs/tx-manager/executor/internal/domain"
	broker "github.com/desync-labs/tx-manager/executor/internal/message-broker/interface"
	pb "github.com/desync-labs/tx-manager/executor/protos/key-manager"
)

const (
	// Topic to execute transactions, for rabbitmq this is exchange name
	execute_topic = "tx_executor"
)

// TODO: naming convention for interface
type ExecutorServiceInterface interface {
	SetupTransactionListener() error
}

type ExecutorService struct {
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

func NewExecutorService(messageBroker broker.MessageBrokerInterface, keyManagerServiceClient pb.KeyManagerServiceClient, priorities []int, ctx context.Context, workerPoolSize int) *ExecutorService {
	ctxExecutorService, cancel := context.WithCancel(ctx)
	ex := &ExecutorService{
		messageBroker:           messageBroker,
		chTransactions:          make(map[int]chan domain.Transaction),
		priorities:              priorities,
		ctx:                     ctxExecutorService,
		cancel:                  cancel,
		workerPoolSize:          workerPoolSize,
		taskQueue:               make(chan domain.Transaction, 100),
		keyManagerServiceClient: keyManagerServiceClient,
	}
	return ex
}

func (e *ExecutorService) SetupTransactionListener() error {
	// Listen for new transactions
	slog.Info("Setting up transaction listener")

	e.mu.Lock()
	defer e.mu.Unlock()

	// Start worker pool to process the transactions as they recieve from message broker
	for i := 0; i < e.workerPoolSize; i++ {
		e.wg.Add(1)
		go e.worker(i + 1)
	}

	for _, p := range e.priorities {
		e.chTransactions[p] = make(chan domain.Transaction)
		go e.listenForNewTransactionForPriority(p, e.chTransactions[p])
	}

	return nil
}

// priority will be p0, p1 or p2
func (e *ExecutorService) listenForNewTransactionForPriority(priority int, chNewTransaction chan domain.Transaction) {
	// Listen for new transactions
	slog.Info("Listening for new transactions", "priority", priority)

	e.messageBroker.ListenForMessages(execute_topic, priority, func(body []byte, ctx context.Context) {
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
			case e.taskQueue <- tx:
			case <-e.ctx.Done():
				return
			}
		case <-e.ctx.Done():
			slog.Info("Shutting down transaction listener", "priority", priority)
			e.closePriorityChannel(priority)
			return
		}
	}
}

func (e *ExecutorService) closePriorityChannel(priority int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if ch, exists := e.chTransactions[priority]; exists {
		close(ch)
		delete(e.chTransactions, priority)
		slog.Info("Closed channel for priority", "priority", priority)
	}
}

func (e *ExecutorService) scheduleTransaction(tx *domain.Transaction) {
	// Receive the transaction
	slog.Info("Execute new transaction", "id", tx.Id)
}

func (e *ExecutorService) Shutdown() {
	slog.Info("Closing Scheduler Service")
	e.cancel()

	e.mu.Lock()
	for p, ch := range e.chTransactions {
		close(ch)
		delete(e.chTransactions, p)
		slog.Info("Closed channel during shutdown", "priority", p)
	}
	e.mu.Unlock()

	slog.Info("Scheduler Service shut down gracefully")
}

// Worker function to process tasks
func (e *ExecutorService) worker(id int) {
	defer e.wg.Done()
	slog.Debug("Worker started", "worker_id", id)
	for {
		select {
		case tx, ok := <-e.taskQueue:
			if !ok {
				slog.Debug("Worker stopping", "worker_id", id)
				return
			}
			e.scheduleTransaction(&tx)
		case <-e.ctx.Done():
			slog.Debug("Worker received shutdown signal", "worker_id", id)
			return
		}
	}
}
