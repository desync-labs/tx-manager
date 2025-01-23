package message_broker

import "context"

type MessageBrokerInterface interface {
	PublishToQueue(queueName string, message float64, ctx context.Context) error
	// Deprecated: Use ListenForSubmitterMessages instead
	ListenForMessages(queueName string, callback func([]byte, context.Context)) error
	ListenForSubmitterMessages(priority string, callback func([]byte, context.Context)) error
	Close() error
	Connect() error
}
