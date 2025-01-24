package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/desync-labs/tx-manager/submitter/internal/domain"
	broker "github.com/desync-labs/tx-manager/submitter/internal/message-broker/interface"
	pb "github.com/desync-labs/tx-manager/submitter/protos/transaction"
	"github.com/go-redis/redis"
)

const (
	// Topic to submit transactions, for rabbitmq this is exchange name
	submit_topic = "tx_submit"
)

type SubmitterServiceInterface interface {
	SubmitTransaction(*pb.TransactionRequest) (string, error)
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
	return tx.Id, s.messageBroker.PublishObject(submit_topic, tx, int(req.GetPriority()), context.Background())
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
