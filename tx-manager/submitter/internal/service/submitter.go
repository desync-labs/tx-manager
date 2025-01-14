package service

import (
	broker "github.com/desync-labs/tx-manager/submitter/internal/message-broker/interface"
)

type SubmitterServiceInterface interface {
	SubmitTransaction(priority int, data []byte, client string) (string, error)
}

// SubmitterService is the service for the submitter
type SubmitterService struct {
	messageBroker broker.MessageBrokerInterface
}

func NewSubmitterService(messageBroker broker.MessageBrokerInterface) *SubmitterService {
	return &SubmitterService{
		messageBroker: messageBroker,
	}
}

func (s *SubmitterService) SubmitTransaction(priority int, data []byte, client string) (string, error) {
	//Generate the unique id for the transaction

	//publish the transaction to the message broker
	return s.messageBroker.Publish(priority, data, client)
}
