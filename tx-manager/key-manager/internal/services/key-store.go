package services

import (
	"encoding/json"
	"fmt"
	"os"

	domain "github.com/desync-labs/tx-manager/key-manager/internal/domain"
)

type KeyStoreInterface interface {
	LoadKeys() (*domain.KeyRecords, error)
}

type KeyStore struct {
	filePath string
}

func NewKeyStore(filePath string) *KeyStore {
	return &KeyStore{
		filePath: filePath,
	}
}

// use 1password sdk to load the keys
func (k *KeyStore) LoadKeys() (*domain.KeyRecords, error) {

	data, err := os.ReadFile(k.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys file: %v", err)
	}

	var jsonKeyRecords domain.JsonKeyRecords
	err = json.Unmarshal(data, &jsonKeyRecords)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal keys JSON: %v", err)
	}

	p1Keys := []domain.KeyRecord{}
	p2Keys := []domain.KeyRecord{}
	p3Keys := []domain.KeyRecord{}

	for _, p1Key := range jsonKeyRecords.P1Keys {
		p1Keys = append(p1Keys,
			domain.KeyRecord{
				PublicKey:             []byte(p1Key),
				AssignedTransactionId: "",
			},
		)
	}

	for _, p2Key := range jsonKeyRecords.P2Keys {
		p2Keys = append(p2Keys,
			domain.KeyRecord{
				PublicKey:             []byte(p2Key),
				AssignedTransactionId: "",
			},
		)
	}

	for _, p3Key := range jsonKeyRecords.P3Keys {
		p3Keys = append(p3Keys,
			domain.KeyRecord{
				PublicKey:             []byte(p3Key),
				AssignedTransactionId: "",
			},
		)
	}

	keys := &domain.KeyRecords{
		P1Keys: p1Keys,
		P2Keys: p2Keys,
		P3Keys: p3Keys,
	}

	return keys, nil
}
