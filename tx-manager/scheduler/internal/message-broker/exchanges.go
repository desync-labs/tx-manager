package message_broker

import "fmt"

const (
	priority1 = "p1"
	priority2 = "p2"
	priority3 = "p3"

	submit_exchange   = "tx_submit"
	schedule_exchange = "tx_schedule"
	executor_exchange = "tx_executor"
)

type Exchange struct {
	exchangeName string
	routingKeys  []string
}

type Exchanges struct {
	Exchanges []*Exchange
}

func InitExchanges() *Exchanges {
	return &Exchanges{
		Exchanges: []*Exchange{
			NewExchange(submit_exchange, []string{priority1, priority2, priority3}),
			NewExchange(schedule_exchange, []string{priority1, priority2, priority3}),
			NewExchange(executor_exchange, []string{priority1, priority2, priority3}),
		},
	}
}

func (exs *Exchanges) GetExchange(exchange string) (Exchange, error) {
	for _, ex := range exs.Exchanges {
		if ex.exchangeName == exchange {
			return *ex, nil
		}
	}
	return Exchange{}, fmt.Errorf("exchange not found")
}

func NewExchange(exchangeName string, routingKeys []string) *Exchange {
	return &Exchange{
		exchangeName: exchangeName,
		routingKeys:  routingKeys,
	}
}

func (e *Exchange) RoutingKey(priority int) (string, error) {
	switch priority {
	case 1:
		return "p1", nil
	case 2:
		return "p2", nil
	case 3:
		return "p3", nil
	}
	return "", fmt.Errorf("priority not found")
}

func (e *Exchange) Queue(routingKey string) string {
	queueName := fmt.Sprintf("%s_%s", e.exchangeName, routingKey)
	return queueName
}

func (r *Exchange) RoutingKeyFromPriority(priority int) (string, error) {
	switch priority {
	case 1:
		return "p1", nil
	case 2:
		return "p2", nil
	case 3:
		return "p3", nil
	default:
		return "", fmt.Errorf("invalid priority %d", priority)
	}
}
