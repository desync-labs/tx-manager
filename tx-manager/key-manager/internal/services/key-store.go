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

// DO NOT COMMIT THIS CODE
func (k *OnePasswordKeyStore) mockKeys() *domain.KeyRecords {
	keys := &domain.KeyRecords{
		P1Keys: []domain.KeyRecord{
			{
				PublicKey:             []byte(""),
				PrivateKey:            []byte(""),
				AssignedTransactionId: "",
			},
		},
		P2Keys: []domain.KeyRecord{
			{
				PublicKey:             []byte(""),
				PrivateKey:            []byte(""),
				AssignedTransactionId: "",
			},
		},
		P3Keys: []domain.KeyRecord{
			{
				PublicKey:             []byte(""),
				PrivateKey:            []byte(""),
				AssignedTransactionId: "",
			},
		},
	}
	return keys
}
