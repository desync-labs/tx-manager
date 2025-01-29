package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	domain "github.com/desync-labs/tx-manager/key-manager/internal/domain"
)

type KeyStoreInterface interface {
	LoadKeys() (*domain.AllKeyRecords, error)
}

type KeyStore struct {
	filePath string
}

func NewKeyStore(filePath string) *KeyStore {
	return &KeyStore{
		filePath: filePath,
	}
}

// Reads and unmarshals the JSON file
func (k *KeyStore) readJsonFile() (domain.AllJsonKeyRecords, error) {
	data, err := ioutil.ReadFile(k.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var allJsonRecords domain.AllJsonKeyRecords
	if err := json.Unmarshal(data, &allJsonRecords); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return allJsonRecords, nil
}

// Transforms AllJsonKeyRecords to AllKeyRecords
func transformRecords(allJson domain.AllJsonKeyRecords) (*domain.AllKeyRecords, error) {
	allKeyRecords := make(domain.AllKeyRecords)

	for networkID, jsonRecords := range allJson {
		keyRecords := &domain.KeyRecords{
			P1Keys: []*domain.KeyRecord{},
			P2Keys: []*domain.KeyRecord{},
			P3Keys: []*domain.KeyRecord{},
		}

		// Process P1Keys
		for _, keyStr := range jsonRecords.P1Keys {
			keyRecords.P1Keys = append(keyRecords.P1Keys, &domain.KeyRecord{
				PublicKey: []byte(keyStr),
			})
		}

		// Process P2Keys
		for _, keyStr := range jsonRecords.P2Keys {
			keyRecords.P2Keys = append(keyRecords.P2Keys, &domain.KeyRecord{
				PublicKey: []byte(keyStr),
			})
		}

		// Process P3Keys
		for _, keyStr := range jsonRecords.P3Keys {
			keyRecords.P3Keys = append(keyRecords.P3Keys, &domain.KeyRecord{
				PublicKey: []byte(keyStr),
			})
		}

		allKeyRecords[networkID] = keyRecords
	}

	return &allKeyRecords, nil
}

func (k *KeyStore) LoadKeys() (*domain.AllKeyRecords, error) {

	kr, err := k.readJsonFile()
	if err != nil {
		return nil, err
	}

	akr, err := transformRecords(kr)
	if err != nil {
		return nil, err
	}

	return akr, nil
}
