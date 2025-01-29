package services

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	domain "github.com/desync-labs/tx-manager/key-manager/internal/domain"
)

var (
	P1 = 1
	P2 = 2
	P3 = 3
)

type KeyManagerServiceInterface interface {
	GetKey(txId string, priority int, ctx context.Context) (publicKey []byte, err error)
	AssignKey(txId string, priority int, ctx context.Context) (err error)
	ReleaseKey(txId string, priority int, ctx context.Context) error
}

type KeyManagerService struct {
	// keyStore        *KeyStore
	keyRecs         *domain.AllKeyRecords
	mu              sync.Mutex
	timeoutDuration time.Duration
	stopChan        chan struct{} // Channel to signal stopping the background goroutine

}

func NewKeyManagerService(keyRecs *domain.AllKeyRecords, timeout time.Duration) (*KeyManagerService, error) {
	kms := &KeyManagerService{
		keyRecs:         keyRecs,
		timeoutDuration: timeout,
		stopChan:        make(chan struct{})}

	go kms.startTimeoutChecker()

	return kms, nil
}

// AssignKey assigns an available key to the given transaction ID based on priority.
// It returns the assigned key and its priority level, or an error if no keys are available.
func (k *KeyManagerService) AssignKey(txId string, priority int, networkId string, ctx context.Context) (err error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	var ok bool = false

	switch priority {
	case P1:
		// Attempt to assign a P1 key first
		ok = k.assignP1Keys(txId, networkId)
		if !ok {
			// If no P1 keys are available, attempt to assign a P2 key
			ok = k.assignP2Keys(txId, networkId)
		}
		if !ok {
			// If no P2 keys are available, attempt to assign a P3 key
			ok = k.assignP3Keys(txId, networkId)
		}
	case P2:
		// Attempt to assign a P2 keys
		ok = k.assignP2Keys(txId, networkId)
		if !ok {
			// If no P2 keys are available, attempt to assign a P2 key
			ok = k.assignP3Keys(txId, networkId)
		}
	case P3:
		// Attempt to assign a P3 keys
		ok = k.assignP3Keys(txId, networkId)
	default:
		return errors.New("invalid priority level")
	}

	if !ok {
		return errors.New("no keys available")
	}

	return nil
}

func (k *KeyManagerService) assignP1Keys(txId, networkId string) bool {
	// Attempt to assign a P1 key first
	keyRec := (*k.keyRecs)[networkId]
	for i := range keyRec.P1Keys {
		if keyRec.P1Keys[i].IsAvailable() {
			keyRec.P1Keys[i].AssignTransaction(txId)
			return true
		}
	}
	return false
}

func (k *KeyManagerService) assignP2Keys(txId, networkId string) bool {
	// Attempt to assign a P2 key first
	keyRec := (*k.keyRecs)[networkId]
	for i := range keyRec.P2Keys {
		if keyRec.P2Keys[i].IsAvailable() {
			keyRec.P2Keys[i].AssignTransaction(txId)
			return true
		}
	}
	return false
}

func (k *KeyManagerService) assignP3Keys(txId, networkId string) bool {
	// Attempt to assign a P3 key first
	keyRec := (*k.keyRecs)[networkId]
	for i := range keyRec.P3Keys {
		if keyRec.P3Keys[i].IsAvailable() {
			keyRec.P3Keys[i].AssignTransaction(txId)
			return true
		}
	}
	return false
}

// GetKey retrieves the private key assigned to the given transaction ID.
// It returns the key and its priority level, or an error if not found.
func (k *KeyManagerService) GetKey(txId string, priority int, networkId string, ctx context.Context) (publicKey []byte, err error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	keyRec := (*k.keyRecs)[networkId]

	if priority == P1 {
		// Search in P1Keys
		for _, keyRecord := range keyRec.P1Keys {
			if keyRecord.AssignedTransactionId == txId {
				return keyRecord.PublicKey, nil
			}
		}
	}

	if priority == P1 || priority == P2 {
		// Search in P2Keys
		for _, keyRecord := range keyRec.P2Keys {
			if keyRecord.AssignedTransactionId == txId {
				return keyRecord.PublicKey, nil
			}
		}
	}

	if priority == P1 || priority == P2 || priority == P3 {
		// Search in P3Keys
		for _, keyRecord := range keyRec.P3Keys {
			if keyRecord.AssignedTransactionId == txId {
				return keyRecord.PublicKey, nil
			}
		}
	}

	return nil, errors.New("no key assigned to the given transaction ID")
}

// ReleaseKey releases the key assigned to the given transaction ID, making it available for reassignment.
func (k *KeyManagerService) ReleaseKey(txId string, priority int, networkId string, ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	keyRec := (*k.keyRecs)[networkId]

	if priority == P1 {
		// Search in P1Keys
		for i := range keyRec.P1Keys {
			if keyRec.P1Keys[i].AssignedTransactionId == txId {
				keyRec.P1Keys[i].UnassignTransaction()
				return nil
			}
		}
	}

	if priority == P1 || priority == P2 {
		// Search in P2Keys
		for i := range keyRec.P2Keys {
			if keyRec.P2Keys[i].AssignedTransactionId == txId {
				keyRec.P2Keys[i].UnassignTransaction()
				return nil
			}
		}
	}

	if priority == P1 || priority == P2 || priority == P3 {
		// Search in P3Keys
		for i := range keyRec.P3Keys {
			if keyRec.P3Keys[i].AssignedTransactionId == txId {
				keyRec.P3Keys[i].UnassignTransaction()
				return nil
			}
		}
	}

	return errors.New("no key found assigned to the given transaction ID")
}

// timeout auto release logic of keys
// startTimeoutChecker starts a background goroutine that periodically checks for expired key assignments.
func (k *KeyManagerService) startTimeoutChecker() {
	//TODO: Set this value as configuration.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			k.releaseExpiredKeys()
		case <-k.stopChan:
			slog.Debug("Stopping timeout checker goroutine.")
			return
		}
	}
}

// releaseExpiredKeys scans all keys and releases those that have exceeded the timeout duration.
func (k *KeyManagerService) releaseExpiredKeys() {
	k.mu.Lock()
	defer k.mu.Unlock()

	now := time.Now()

	for networkId := range *k.keyRecs {

		keyRec := (*k.keyRecs)[networkId]

		// Check P1Keys
		for i := range keyRec.P1Keys {
			keyRecord := keyRec.P1Keys[i]
			if !keyRecord.IsAvailable() && now.Sub(keyRecord.AssignedAt) > k.timeoutDuration {
				slog.Info("Releasing expired key: %s", "Priority", "P1", "tx-id", keyRecord.AssignedTransactionId)
				keyRecord.UnassignTransaction()
			}
		}

		// Check P2Keys
		for i := range keyRec.P2Keys {
			keyRecord := keyRec.P2Keys[i]
			if !keyRecord.IsAvailable() && now.Sub(keyRecord.AssignedAt) > k.timeoutDuration {
				slog.Info("Releasing expired key: %s", "Priority", "P2", "tx-id", keyRecord.AssignedTransactionId)
				keyRecord.UnassignTransaction()
			}
		}

		// Check P3Keys
		for i := range keyRec.P3Keys {
			keyRecord := keyRec.P3Keys[i]
			if !keyRecord.IsAvailable() && now.Sub(keyRecord.AssignedAt) > k.timeoutDuration {
				slog.Info("Releasing expired key: %s", "Priority", "P3", "tx-id", keyRecord.AssignedTransactionId)
				keyRecord.UnassignTransaction()
			}
		}
	}

}

// Stop stops the background timeout checker goroutine.
func (k *KeyManagerService) Stop() {
	slog.Info("Stopping key manager service.")
	close(k.stopChan)
}
