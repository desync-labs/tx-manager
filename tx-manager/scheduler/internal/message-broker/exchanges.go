package message_broker

import "fmt"

type Exchange struct {
	exchangeName string
	routingKeys  []string
}

func (e *Exchange) QueueName(priority string) string {
	return fmt.Sprintf("%s_%s", e.exchangeName, priority)
}

type Exchanges struct {
	Exchanges []*Exchange
}

func InitExchanges() *Exchanges {
	return &Exchanges{
		Exchanges: []*Exchange{
			NewExchange("tx_submit", []string{"p1", "p2", "p3"}),
			NewExchange("tx_process", []string{"p1", "p2", "p3"}),
			NewExchange("tx_processed", []string{"p1", "p2", "p3"}),
		},
	}
}

func NewExchange(exchangeName string, routingKeys []string) *Exchange {
	return &Exchange{
		exchangeName: exchangeName,
		routingKeys:  routingKeys,
	}
}

func (e *Exchange) Queue(routingKey string) string {
	queueName := fmt.Sprintf("%s_%s", e.exchangeName, routingKey)
	return queueName
}
