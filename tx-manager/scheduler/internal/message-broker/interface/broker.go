package message_broker

import "context"

type MessageBrokerInterface interface {
	//If priority is -1, then message will be published to default routing key of the exchange
	Publish(topic string, data string, priority int, ctx context.Context) error
	//If priority is -1, then message will be published to default routing key of the exchange
	PublishObject(topic string, data interface{}, priority int, ctx context.Context) error

	ListenForMessages(topic string, priority int, callback func([]byte, context.Context)) error
}
