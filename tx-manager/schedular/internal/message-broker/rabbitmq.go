package message_broker

import (
	"context"
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

	if err := r.Connect(); err != nil {
		return nil, err
	}

	// exchanges := Exchanges{}
	r.exchanges = InitExchanges()

	return r, nil
}

// connect establishes a connection to RabbitMQ
func (r *RabbitMQ) Connect() error {
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

// PublishToQueue publishes a float64 value to a specific queue.
// If the queue does not exist, it creates a durable queue.
func (r *RabbitMQ) PublishToQueue(queueName string, message float64, ctx context.Context) error {

	_, err := r.ch.QueueDeclare(
		queueName, // Queue name
		true,      // Durable
		false,     // Delete when unused
		false,     // Exclusive
		false,     // No-wait
		nil,       // Arguments
	)
	if err != nil {
		return err
	}

	err = r.ch.Publish(
		"",        // Exchange
		queueName, // Routing key (queue name)
		false,     // Mandatory
		false,     // Immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("%f", message)),
		},
	)
	return err
}

// PublishToQueue publishes a key-value to a specific queue.
// If the queue does not exist, it creates a durable queue.
// TODO: Make a generic method to publish any type of message
func (r *RabbitMQ) PublishObjectToQueue(queueName string, message string, ctx context.Context) error {
	_, err := r.ch.QueueDeclare(
		queueName, // Queue name
		true,      // Durable
		false,     // Delete when unused
		false,     // Exclusive
		false,     // No-wait
		nil,       // Arguments
	)
	if err != nil {
		return err
	}

	err = r.ch.Publish(
		"",        // Exchange
		queueName, // Routing key (queue name)
		false,     // Mandatory
		false,     // Immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("%s", message)),
		},
	)
	return err
}

// ListenForMessages listens for incoming messages on a specific queue and triggers the callback.
// Deprecated: Use ListenForSubmitterMessages instead
func (r *RabbitMQ) ListenForMessages(queueName string, callback func([]byte, context.Context)) error {

	msgs, err := r.ch.Consume(
		queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			callback(msg.Body, r.ctx)
		}
	}()

	return nil
}

func (r *RabbitMQ) ListenForSubmitterMessages(priority string, callback func([]byte, context.Context)) error {

	submitterExchange := r.exchanges.Exchanges[0]
	err := r.listenForMessages(submitterExchange.exchangeName, submitterExchange.QueueName(priority), priority, callback)
	if err != nil {
		slog.Error("Failed to listen for submitter messages", "error", err)
		return err
	}

	return nil
}

// ListenForMessages listens for incoming messages on a specific exchange,
// binds the queue to that exchange with the given routing key, and triggers the callback.
func (r *RabbitMQ) listenForMessages(exchangeName, queueName, routingKey string, callback func([]byte, context.Context)) error {
	// Declare the exchange
	err := r.ch.ExchangeDeclare(
		exchangeName, // name
		"direct",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange %s: %w", exchangeName, err)
	}

	// Declare the queue
	queue, err := r.ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	// Bind the queue to the exchange with the routing key
	err = r.ch.QueueBind(
		queue.Name,   // queue name
		routingKey,   // routing key
		exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue %s to exchange %s with routing key %s: %w",
			queueName, exchangeName, routingKey, err)
	}

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

// PublishObjectToExchange publishes a message to a specific exchange in fanout mode.
func (r *RabbitMQ) PublishObjectToExchange(exchangeName string, message string, ctx context.Context) error {

	// Declare a fanout exchange
	err := r.ch.ExchangeDeclare(
		exchangeName, // Exchange name
		"fanout",     // Type of exchange
		true,         // Durable
		false,        // Auto-deleted
		false,        // Internal
		false,        // No-wait
		nil,          // Arguments
	)
	if err != nil {
		// span.SetStatus(codes.Error, err.Error())
		// span.RecordError(err)
		return err
	}

	// Publish the message to the fanout exchange
	err = r.ch.Publish(
		exchangeName, // Exchange name
		"",           // Routing key (empty for fanout exchange)
		false,        // Mandatory
		false,        // Immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		},
	)

	return err
}

// Close closes the RabbitMQ connection and channel.
func (r *RabbitMQ) Close() error {

	err := r.ch.Close()
	if err != nil {
		return err
	}

	return r.conn.Close()
}
