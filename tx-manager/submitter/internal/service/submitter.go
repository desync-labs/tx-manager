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
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SubmitterServiceInterface interface {
	SubmitTransaction(*pb.TransactionRequest) (string, error)
	SetupTransactionStatusEvent() error
	SetupTransactionStatusListener(txID string, statusCh chan<- *pb.TransactionStatusUpdate) error
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
		slog.Error("Failed to create transaction", "error", err)
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
		}
	})

	return nil
}

func (s *SubmitterService) SetupTransactionStatusListener(txID string, statusCh chan<- *pb.TransactionStatusUpdate) error {
	slog.Info("Setting up transaction status listener", "tx-id", txID)

	// Listen for messages from the message broker
	err := s.messageBroker.ListenForMessages(mb.Tx_Status_Exchange, -1, func(body []byte, ctx context.Context) {
		txStatus := &domain.TransactionStatus{}
		err := json.Unmarshal(body, txStatus)
		if err != nil {
			slog.Error("Failed to unmarshal message", "error", err)
			return
		}

		// Filter messages for the specific transaction ID
		if txStatus.Id != txID {
			return
		}

		tx_hash, ok := txStatus.Metadata["txHash"]
		if !ok {
			tx_hash = ""
		}

		// Map domain.TransactionStatus to pb.TransactionStatusUpdate
		pbStatus := &pb.TransactionStatusUpdate{
			TxKey:     txStatus.Id,
			Status:    mapDomainStatusToProto(txStatus.Status),
			Message:   txStatus.Response,
			Timestamp: timestamppb.New(txStatus.At),
			TxHash:    tx_hash,
		}

		// Send the status update to the channel
		select {
		case statusCh <- pbStatus:
		case <-ctx.Done():
			slog.Info("Context done, stopping status listener:", "tx-id", txID)
		}
	})

	if err != nil {
		slog.Error("Failed to set up transaction status listener", "error", err)
		return err
	}

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

// mapDomainStatusToProto maps domain.Tx_Status to pb.TransactionStatus
func mapDomainStatusToProto(status int) pb.TransactionStatus {
	switch status {
	case domain.Tx_Status_None:
		return pb.TransactionStatus_NONE
	case domain.Tx_Status_Submitted:
		return pb.TransactionStatus_SUBMITTED
	case domain.Tx_Status_Scheduled:
		return pb.TransactionStatus_SCHEDULED
	case domain.Tx_Status_Executing:
		return pb.TransactionStatus_EXECUTING
	case domain.Tx_Status_Confirmed:
		return pb.TransactionStatus_CONFIRMED
	case domain.Tx_Status_TimedOut:
		return pb.TransactionStatus_TIMEDOUT
	case domain.Tx_Status_Error:
		return pb.TransactionStatus_ERROR
	default:
		return pb.TransactionStatus_NONE
	}
}
