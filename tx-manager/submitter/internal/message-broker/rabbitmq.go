package message_broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/streadway/amqp"
)

// RabbitMQ implements the MessageBroker interface for RabbitMQ.
type RabbitMQ struct {
	conn      *amqp.Connection
	ch        *amqp.Channel
	ctx       context.Context
	amqpURI   string
	exchanges *Exchanges
}

func NewRabbitMQ(amqpURI string, ctx context.Context) (*RabbitMQ, error) {
	r := &RabbitMQ{
		ctx:     ctx,
		amqpURI: amqpURI,
	}

	if err := r.connect(); err != nil {
		return nil, err
	}

	// exchanges := Exchanges{}
	r.exchanges = InitExchanges()
	r.setupRouting()

	return r, nil
}

// PublishToQueue publishes a float64 value to a specific queue.
// If the queue does not exist, it creates a durable queue.
func (r *RabbitMQ) Publish(exchange string, data string, priority int, ctx context.Context) error {

	ex, err := r.exchanges.GetExchange(exchange)
	if err != nil {
		return fmt.Errorf("failed to get exhange: %w", err)
	}

	rk, err := ex.RoutingKey(priority)

	if err != nil {
		return fmt.Errorf("failed to get routing key from priority: %w", err)
	}

	// Publish the message to the appropriate queue
	err = r.ch.Publish(
		exchange, // Default exchange
		rk,       // The routing key (queue name)
		false,    // Mandatory
		false,    // Immediate
		amqp.Publishing{
			ContentType: "text/plain", // Using application/json for content type
			Body:        []byte(fmt.Sprintf("%s", data)),
			Priority:    uint8(priority), // The serialized JSON object
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	slog.Debug("Publishing message to message broker",
		"exxhange", exchange,
		"message", data,
	)

	return nil

}

func (r *RabbitMQ) PublishObject(exchange string, data interface{}, priority int, ctx context.Context) error {

	ex, err := r.exchanges.GetExchange(exchange)
	if err != nil {
		return fmt.Errorf("failed to get exhange: %w", err)
	}

	rk, err := ex.RoutingKey(priority)

	if err != nil {
		return fmt.Errorf("failed to get routing key from priority: %w", err)
	}

	// Serialize the provided data (which can be any JSON object)
	messageData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize message data: %w", err)
	}

	// Publish the message to the appropriate queue
	err = r.ch.Publish(
		ex.exchangeName, // Default exchange
		rk,              // The routing key (queue name)
		false,           // Mandatory
		false,           // Immediate
		amqp.Publishing{
			ContentType: "application/json", // Using application/json for content type
			Body:        messageData,
			Priority:    uint8(priority), // The serialized JSON object
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	slog.Debug("Publishing message to message broker",
		"exxhange", exchange,
		"message", messageData,
	)

	return nil
}

// ListenForMessages listens for incoming messages on a specific exchange,
// binds the queue to that exchange with the given routing key, and triggers the callback.
func (r *RabbitMQ) ListenForMessages(exchangeName string, priority int, callback func([]byte, context.Context)) error {

	ex, err := r.exchanges.GetExchange(exchangeName)
	if err != nil {
		return err
	}

	queueName := ex.QueueName(priority)

	// Start consuming messages from the queue
	msgs, err := r.ch.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack (set to false for manual ack)
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer for queue %s: %w", queueName, err)
	}

	// Handle messages in a separate goroutine
	go func() {
		for {
			select {
			case <-r.ctx.Done():
				slog.Info("Shutting down consumer", "queue", queueName)
				return
			case msg, ok := <-msgs:
				if !ok {
					slog.Info("Message channel closed", "queue", queueName)
					return
				}

				// Trigger the callback with message body and context
				callback(msg.Body, r.ctx)

				// Acknowledge the message after processing
				if err := msg.Ack(false); err != nil {
					slog.Error("Failed to acknowledge message", "queue", queueName, "error", err)
				}
			}
		}
	}()

	return nil
}

// Close closes the RabbitMQ connection and channel.
func (r *RabbitMQ) Close() {
	if r.ch != nil {
		r.ch.Close()
		slog.Debug("RabbitMQ channel closed")
	}
	if r.conn != nil {
		r.conn.Close()
		slog.Debug("RabbitMQ connection closed")
	}
}

// connect establishes a connection to RabbitMQ
func (r *RabbitMQ) connect() error {
	conn, err := amqp.Dial(r.amqpURI)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()

	if err != nil {
		conn.Close()
		return err
	}

	r.conn = conn
	r.ch = ch
	return nil
}

func (r *RabbitMQ) setupRouting() error {

	for _, ex := range r.exchanges.Exchanges {
		// Declare the exchange
		err := r.ch.ExchangeDeclare(
			ex.exchangeName, // name
			"direct",        // type
			true,            // durable
			false,           // auto-deleted
			false,           // internal
			false,           // no-wait
			nil,             // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", ex.exchangeName, err)
		}

		//Setu queue for each routing key
		for _, rk := range ex.routingKeys {
			// Declare the queue
			queue, err := r.ch.QueueDeclare(
				rk,    // name
				true,  // durable
				false, // delete when unused
				false, // exclusive
				false, // no-wait
				nil,   // arguments
			)
			if err != nil {
				return fmt.Errorf("failed to declare queue %s: %w", rk, err)
			}

			// Bind the queue to the exchange with the routing key
			err = r.ch.QueueBind(
				queue.Name,      // queue name
				rk,              // routing key
				ex.exchangeName, // exchange
				false,
				nil,
			)
			if err != nil {
				return fmt.Errorf("failed to bind queue to exchange %s with routing key %s: %w",
					ex.exchangeName, rk, err)
			}
		}
	}

	return nil
}
