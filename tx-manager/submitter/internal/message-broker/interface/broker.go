package message_broker

type MessageBrokerInterface interface {
	Publish(priority int, data []byte, client string) (string, error)
}
