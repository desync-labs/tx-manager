package domain

import (
	"fmt"
	"time"
)

// Transaction is a domain model for a transaction
type Transaction struct {
	Id          string
	AppName     string
	Priority    int
	Data        []byte
	SubmittedAt time.Time
}

func NewTransaction(appName string, priority int, data []byte) (*Transaction, error) {

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

	return &Transaction{
		AppName:     appName,
		Priority:    priority,
		Data:        data,
		SubmittedAt: time.Now(),
	}, nil
}
