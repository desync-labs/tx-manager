package domain

import (
	"fmt"
	"time"
)

// Transaction is a domain model for a transaction
type Transaction struct {
	Id              string
	AppName         string
	Priority        int
	NetworkID       int
	ContractAddress string
	Data            []byte
	SubmittedAt     time.Time
}

func NewTransaction(appName string, priority int, network_id int, contract_addr string, data []byte) (*Transaction, error) {

	//Validate and prepare the transaction data
	//validate the request
	//TODO: Add ACL to only allow from valid applications
	if appName == "" {
		return nil, fmt.Errorf("app name is required")
	}

	if priority < 1 || priority > 3 {
		return nil, fmt.Errorf("Invalid priority value")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("Transaction data is required")
	}

	if network_id == 0 {
		return nil, fmt.Errorf("Network id is required")
	}

	if len(contract_addr) == 0 {
		return nil, fmt.Errorf("Contract address is required")
	}

	return &Transaction{
		AppName:         appName,
		Priority:        priority,
		NetworkID:       network_id,
		ContractAddress: contract_addr,
		Data:            data,
		SubmittedAt:     time.Now(),
	}, nil
}
