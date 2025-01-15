package message_broker

type MessageBrokerInterface interface {
	Publish(priority int, data interface{}) error
}
