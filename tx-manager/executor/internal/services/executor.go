package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	cache "github.com/desync-labs/tx-manager/executor/internal/cache"
	"github.com/desync-labs/tx-manager/executor/internal/domain"
	mb "github.com/desync-labs/tx-manager/executor/internal/message-broker"
	broker "github.com/desync-labs/tx-manager/executor/internal/message-broker/interface"
	pb "github.com/desync-labs/tx-manager/executor/protos/key-manager"
)

type ExecutorServiceInterface interface {
	SetupTransactionListener() error
}

type TransactionExecutorInterface interface {
	Execute(key string, tx *domain.Transaction) (bool, string, error)
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
	keyCache                cache.KeyCacheInterface

	// map of transaction executor, where key is network id
	registeredTxExecutors map[int]TransactionExecutorInterface
}

func NewExecutorService(messageBroker broker.MessageBrokerInterface, keyManagerServiceClient pb.KeyManagerServiceClient, keyCache cache.KeyCacheInterface, priorities []int, ctx context.Context, workerPoolSize int) *ExecutorService {
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
		keyCache:                keyCache,
		registeredTxExecutors:   make(map[int]TransactionExecutorInterface),
	}

	return ex
}

func (e *ExecutorService) RegisterTransactionExecutor(networkID int, txExecutor TransactionExecutorInterface) {
	e.registeredTxExecutors[networkID] = txExecutor
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

	e.messageBroker.ListenForMessages(mb.Executor_Exchange, priority, func(body []byte, ctx context.Context) {
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

func (e *ExecutorService) processTransaction(tx *domain.Transaction) {
	// Receive the transaction
	slog.Info("Processing transaction", "id", tx.Id)

	//Fetch the key from key manager
	key, err := e.keyManagerServiceClient.GetKey(e.ctx, &pb.KeyManagerRequest{
		TxId:      tx.Id,
		Priority:  pb.Priority(tx.Priority),
		NetworkId: strconv.Itoa(tx.NetworkID),
	})

	//Todo: add the transaction to DLQ
	if err != nil {
		slog.Error("Failed to get key from key manager", "error", err)
		return
	}

	// Execute the transaction
	slog.Info("Transaction executed with public key", "id", tx.Id)
	e.executingTransaction(key.Key, tx)
}

func (e *ExecutorService) executingTransaction(key string, tx *domain.Transaction) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	ex, ok := e.registeredTxExecutors[tx.NetworkID]
	if !ok {
		slog.Error("No transaction executor found for network", "network_id", tx.NetworkID)
		e.publishTransactionEvent(tx.Id, domain.Tx_Status_Error, fmt.Sprintf("No transaction executor found for network %d", tx.NetworkID), nil)
		return nil
	}

	//Fetch the private key
	privateKey, err := e.fetchPrivateKey(key)
	if err != nil {
		slog.Error("Failed to fetch private key", "public-key", key, "error", err)
		e.publishTransactionEvent(tx.Id, domain.Tx_Status_Error, err.Error(), nil)
		return err
	}

	e.publishTransactionEvent(tx.Id, domain.Tx_Status_Executing, "Transaction is being executed", nil)

	success, txHash, err := ex.Execute(privateKey, tx)
	if err != nil {
		slog.Error("Failed to execute transaction", "error", err)

		metaData := make(map[string]string)
		if txHash != "" {
			metaData["tx_hash"] = txHash
		}

		e.publishTransactionEvent(tx.Id, domain.Tx_Status_Error, err.Error(), metaData)
		return err
	}

	if success {
		slog.Info("Transaction executed successfully", "id", tx.Id)

		metaData := make(map[string]string)
		if txHash != "" {
			metaData["tx_hash"] = txHash
		}

		e.publishTransactionEvent(tx.Id, domain.Tx_Status_Confirmed, "Transaction executed successfully", metaData)
		return nil

	}

	return nil
}

func (e *ExecutorService) publishTransactionEvent(id string, status int, response string, txMetaData map[string]string) {
	// Publish transaction status to message broker
	tx_status := &domain.TransactionStatus{
		Id:       id,
		Status:   status,
		At:       time.Now(),
		Response: response,
	}

	if txMetaData != nil {
		tx_status.Metadata = txMetaData
	}

	e.messageBroker.PublishObject(mb.Tx_Status_Exchange, tx_status, -1, context.Background())
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
			e.processTransaction(&tx)
		case <-e.ctx.Done():
			slog.Debug("Worker received shutdown signal", "worker_id", id)
			return
		}
	}
}

func (e *ExecutorService) fetchPrivateKey(publicKey string) (string, error) {
	slog.Debug("Fetching private key", "public-key", publicKey)

	cacheKey, err := e.keyCache.Get(publicKey)
	if err != nil {
		return "", err
	}

	return cacheKey, nil
}
