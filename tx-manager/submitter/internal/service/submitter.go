package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/desync-labs/tx-manager/submitter/internal/domain"
	mb "github.com/desync-labs/tx-manager/submitter/internal/message-broker"
	broker "github.com/desync-labs/tx-manager/submitter/internal/message-broker/interface"
	pb "github.com/desync-labs/tx-manager/submitter/protos/transaction"
	"github.com/go-redis/redis"
)

type SubmitterServiceInterface interface {
	SubmitTransaction(*pb.TransactionRequest) (string, error)
	SetupTransactionStatusEvent() error
}

// SubmitterService is the service for the submitter
type SubmitterService struct {
	messageBroker broker.MessageBrokerInterface
	redisClient   *redis.Client
}

func NewSubmitterService(messageBroker broker.MessageBrokerInterface, redisClient *redis.Client) *SubmitterService {
	return &SubmitterService{
		messageBroker: messageBroker,
		redisClient:   redisClient,
	}
}

func (s *SubmitterService) SubmitTransaction(req *pb.TransactionRequest) (string, error) {
	//Generate the unique id for the transaction

	tx, err := domain.NewTransaction(req.GetAppName(),
		int(req.GetPriority()),
		int(req.GetNetwork()),
		string(req.GetContractAddress()),
		[]byte(req.GetTxData()))

	if err != nil {
		slog.Error("Failed to create transaction: %v", err)
		return "", err
	}

	// Generate unique transaction ID using Redis
	txID, err := s.generateTransactionID(tx.AppName)
	if err != nil {
		return "", fmt.Errorf("failed to generate transaction ID: %w", err)
	}

	tx.Id = txID

	//publish the transaction to the message broker
	err = s.messageBroker.PublishObject(mb.Submit_Exchange, tx, int(req.GetPriority()), context.Background())
	if err != nil {
		slog.Error("Failed to publish transaction to message broker", "error", err)
		return "", err
	}

	// Publish transaction status to message broker
	tx_status := &domain.TransactionStatus{
		Id:       tx.Id,
		Status:   domain.Tx_Status_Submitted,
		At:       time.Now(),
		Response: "Transaction submitted for execution",
	}
	s.messageBroker.PublishObject(mb.Tx_Status_Exchange, tx_status, -1, context.Background())

	return txID, nil
}

func (s *SubmitterService) SetupTransactionStatusEvent() error {
	// Listen for new transactions
	slog.Info("Setting up transaction status listener")

	s.messageBroker.ListenForMessages(mb.Tx_Status_Exchange, -1, func(body []byte, ctx context.Context) {
		txStatus := &domain.TransactionStatus{}
		err := json.Unmarshal(body, txStatus)
		if err != nil {
			slog.Error("Failed to unmarshal message", "error", err)
			return
		} else {
			slog.Info("Received transaction status", "id", txStatus.Id, "status", txStatus.Status, "response", txStatus.Response, "metadata", txStatus.Metadata)
		}
	})

	return nil
}

// generateTransactionID generates a unique transaction ID using Redis INCR command
func (s *SubmitterService) generateTransactionID(appName string) (string, error) {
	// Generate a unique ID using Redis' INCR command
	txNumber, err := s.redisClient.Incr("transaction_counter").Result()
	if err != nil {
		return "", fmt.Errorf("failed to increment transaction counter in Redis: %w", err)
	}

	// Combine app name and incremented value to form a transaction ID
	txID := fmt.Sprintf("%s-%d", appName, txNumber)
	return txID, nil
}
