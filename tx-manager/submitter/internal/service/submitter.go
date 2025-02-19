package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
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

	SubscribeTxStatus(txID string) (<-chan *pb.TransactionStatusUpdate, func())
	SetupGlobalTxStatusListener() error
}

// SubmitterService is the service for the submitter
type SubmitterService struct {
	messageBroker broker.MessageBrokerInterface
	redisClient   *redis.Client
	txSubscribers sync.Map
}

func NewSubmitterService(messageBroker broker.MessageBrokerInterface, redisClient *redis.Client) *SubmitterService {
	return &SubmitterService{
		messageBroker: messageBroker,
		redisClient:   redisClient,
	}
}

// SetupGlobalTxStatusListener is intended to be called once during startup.
// It listens for messages from the broker and dispatches them to subscribers.
func (s *SubmitterService) SetupGlobalTxStatusListener() error {
	slog.Info("Setting up global transaction status listener")
	err := s.messageBroker.ListenForMessages(mb.Tx_Status_Exchange, -1, func(body []byte, ctx context.Context) {
		txStatus := &domain.TransactionStatus{}
		err := json.Unmarshal(body, txStatus)
		if err != nil {
			slog.Error("Failed to unmarshal message", "error", err)
			return
		}

		// Map domain.TransactionStatus to pb.TransactionStatusUpdate
		txHash, ok := txStatus.Metadata["txHash"]
		if !ok {
			txHash = ""
		}
		pbStatus := &pb.TransactionStatusUpdate{
			TxKey:     txStatus.Id,
			Status:    mapDomainStatusToProto(txStatus.Status),
			Message:   txStatus.Response,
			Timestamp: timestamppb.New(txStatus.At),
			TxHash:    txHash,
		}

		// Dispatch the message to subscribers registered for this txID.
		if subs, ok := s.txSubscribers.Load(txStatus.Id); ok {
			if channels, ok := subs.([]chan *pb.TransactionStatusUpdate); ok {
				for _, ch := range channels {
					select {
					case ch <- pbStatus:
					default:
						slog.Warn("Subscriber channel full", "tx-id", txStatus.Id)
					}
				}
			}
		} else {
			slog.Debug("No subscribers for tx status", "tx-id", txStatus.Id)
		}
	})
	if err != nil {
		slog.Error("Failed to set up global transaction status listener", "error", err)
		return err
	}
	return nil
}

// SubscribeTxStatus registers a subscriber channel for a specific txID.
// Returns the channel on which status updates will be sent and an unsubscribe function.
func (s *SubmitterService) SubscribeTxStatus(txID string) (<-chan *pb.TransactionStatusUpdate, func()) {
	ch := make(chan *pb.TransactionStatusUpdate, 10) // buffered channel
	// Load or create an entry for this txID.
	val, _ := s.txSubscribers.LoadOrStore(txID, []chan *pb.TransactionStatusUpdate{ch})
	if channels, ok := val.([]chan *pb.TransactionStatusUpdate); ok {
		// If an entry already exists, append the new channel.
		s.txSubscribers.Store(txID, append(channels, ch))
	}

	// Return an unsubscribe function to remove this channel.
	unsubscribe := func() {
		if val, ok := s.txSubscribers.Load(txID); ok {
			if channels, ok := val.([]chan *pb.TransactionStatusUpdate); ok {
				newChannels := make([]chan *pb.TransactionStatusUpdate, 0, len(channels))
				for _, c := range channels {
					if c != ch {
						newChannels = append(newChannels, c)
					}
				}
				if len(newChannels) == 0 {
					s.txSubscribers.Delete(txID)
				} else {
					s.txSubscribers.Store(txID, newChannels)
				}
			}
		}
		close(ch)
	}
	return ch, unsubscribe
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
			slog.Warn("Ignoring status update for different transaction", "tx-id", txID, "status-tx-id", txStatus.Id)
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
