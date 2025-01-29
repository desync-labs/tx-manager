package message_broker

import "fmt"

const (
	priority1 = "p1"
	priority2 = "p2"
	priority3 = "p3"

	txStatusEvent = "event"

	Submit_Exchange    = "tx_submit"
	Schedule_Exchange  = "tx_schedule"
	Executor_Exchange  = "tx_executor"
	Tx_Status_Exchange = "tx_status"
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
			NewExchange(Submit_Exchange, []string{priority1, priority2, priority3}),
			NewExchange(Schedule_Exchange, []string{priority1, priority2, priority3}),
			NewExchange(Executor_Exchange, []string{priority1, priority2, priority3}),
			NewExchange(Tx_Status_Exchange, []string{txStatusEvent}),
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

// If priority is -1, it returns the first routing key
func (e *Exchange) RoutingKey(priority int) (string, error) {
	switch priority {
	case -1:
		return e.routingKeys[0], nil
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

func (e *Exchange) RoutingKeyFromPriority(priority int) (string, error) {
	switch priority {
	case -1:
		return e.routingKeys[0], nil
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
