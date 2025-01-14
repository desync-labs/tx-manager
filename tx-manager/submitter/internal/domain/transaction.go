package domain

import "time"

// Transaction is a domain model for a transaction
type Transaction struct {
	Id          string
	AppName     string
	Priority    int
	Data        []byte
	SubmittedAt time.Time
}
