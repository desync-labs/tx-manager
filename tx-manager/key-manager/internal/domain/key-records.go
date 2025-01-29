package domain

import (
	"time"
)

type AllKeyRecords map[string]*KeyRecords
type AllJsonKeyRecords map[string]JsonKeyRecords

type JsonKeyRecords struct {
	P1Keys []string `json:"P1Keys"`
	P2Keys []string `json:"P2Keys"`
	P3Keys []string `json:"P3Keys"`
}

type KeyRecords struct {
	P1Keys []*KeyRecord
	P2Keys []*KeyRecord
	P3Keys []*KeyRecord
}

type KeyRecord struct {
	PublicKey             []byte    `json:"PublicKey"`
	AssignedTransactionId string    `json:"AssignedTransactionId,omitempty"`
	AssignedAt            time.Time `json:"AssignedAt,omitempty"`
}

// Check if the key is available
func (k *KeyRecord) IsAvailable() bool {
	return len(k.AssignedTransactionId) == 0
}

// Assign a transaction to the key
func (k *KeyRecord) AssignTransaction(txId string) {
	k.AssignedTransactionId = txId
	k.AssignedAt = time.Now()
}

// Unassign a transaction from the key
func (k *KeyRecord) UnassignTransaction() {
	k.AssignedTransactionId = ""
}
