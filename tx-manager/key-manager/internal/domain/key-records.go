package domain

import "time"

type KeyRecords struct {
	P1Keys []KeyRecord
	P2Keys []KeyRecord
	P3Keys []KeyRecord
}

type KeyRecord struct {
	PublicKey             []byte
	AssignedTransactionId string
	AssignedAt            time.Time // Timestamp when the key was assigned
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
