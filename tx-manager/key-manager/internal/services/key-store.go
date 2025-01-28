package services

import domain "github.com/desync-labs/tx-manager/key-manager/internal/domain"

type KeyStoreInterface interface {
	LoadKeys() (*domain.KeyRecords, error)
}

type OnePasswordKeyStore struct {
	vaultId      string
	secret_token string
}

func NewKeyStore(vaultId string, secret_token string) *OnePasswordKeyStore {
	return &OnePasswordKeyStore{
		vaultId:      vaultId,
		secret_token: secret_token,
	}
}

// use 1password sdk to load the keys
func (k *OnePasswordKeyStore) LoadKeys() (*domain.KeyRecords, error) {
	return k.mockKeys(), nil
}

// TODO: Load this configuration from local config file.
func (k *OnePasswordKeyStore) mockKeys() *domain.KeyRecords {
	keys := &domain.KeyRecords{
		P1Keys: []domain.KeyRecord{
			{
				PublicKey:             []byte("0xDe47c8c7dfA5FD603b0C0813C19c268391B7857f"),
				AssignedTransactionId: "",
			},
		},
		P2Keys: []domain.KeyRecord{
			{
				PublicKey:             []byte("0x29Ba7FE646498afB6913AFf7B774C1D8B3802713"),
				AssignedTransactionId: "",
			},
		},
		P3Keys: []domain.KeyRecord{
			{
				PublicKey:             []byte("0x5d758B0BEF03Ec903a0AD961381D84A02aCC3a34"),
				AssignedTransactionId: "",
			},
		},
	}
	return keys
}
