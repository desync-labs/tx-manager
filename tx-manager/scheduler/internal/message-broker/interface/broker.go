package message_broker

import "context"

type MessageBrokerInterface interface {
	Publish(topic string, data string, priority int, ctx context.Context) error
	PublishObject(topic string, data interface{}, priority int, ctx context.Context) error
	ListenForMessages(topic string, priority int, callback func([]byte, context.Context)) error
}
