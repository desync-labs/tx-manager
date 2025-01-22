package message_broker

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/streadway/amqp"
)

// TODO: can be moved to common package
const (
	priority1       = "p1"
	priority2       = "p2"
	priority3       = "p3"
	submit_exchange = "tx_submit"
)

// RabbitMQPublisher is the implementation of MessageBrokerInterface using RabbitMQ
type RabbitMQPublisher struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

// NewRabbitMQPublisher creates a new RabbitMQPublisher instance
func NewRabbitMQPublisher(amqpURI string) (*RabbitMQPublisher, error) {
	// Establish connection to RabbitMQ
	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Create a channel for communication with RabbitMQ
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare priority queues (p1, p2, p3)
	p1Queue := fmt.Sprintf("%s_%s", submit_exchange, priority1)
	_, err = ch.QueueDeclare(p1Queue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare p1 queue: %w", err)
	}

	p2Queue := fmt.Sprintf("%s_%s", submit_exchange, priority2)
	_, err = ch.QueueDeclare(p2Queue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare p2 queue: %w", err)
	}

	p3Queue := fmt.Sprintf("%s_%s", submit_exchange, priority3)
	_, err = ch.QueueDeclare(p3Queue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare p3 queue: %w", err)
	}

	err = ch.ExchangeDeclare(
		submit_exchange, // name
		"direct",        // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		return nil, err
	}

	// Return the RabbitMQPublisher instance
	return &RabbitMQPublisher{
		connection: conn,
		channel:    ch,
	}, nil
}

// TODO: why we are sending priority while publishing the message
// Publish publishes the transaction to the appropriate RabbitMQ queue based on priority
func (r *RabbitMQPublisher) Publish(priority int, data interface{}) error {
	var routingKey string

	// Determine the queue based on priority
	switch priority {
	case 1:
		routingKey = priority1
	case 2:
		routingKey = priority2
	case 3:
		routingKey = priority3
	default:
		return fmt.Errorf("invalid priority %d", priority)
	}

	queueName := fmt.Sprintf("%s_%s", submit_exchange, routingKey)

	// Serialize the provided data (which can be any JSON object)
	messageData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize message data: %w", err)
	}

	// Bind the queue to the exchange with the routing key
	err = r.channel.QueueBind(
		queueName,       // queue name
		routingKey,      // routing key
		submit_exchange, // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue %s to exchange %s with routing key %s: %w",
			queueName, submit_exchange, routingKey, err)
	}

	// Publish the message to the appropriate queue
	err = r.channel.Publish(
		submit_exchange, // Default exchange
		routingKey,      // The routing key (queue name)
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
		"queue", queueName,
		"message", messageData,
	)

	return nil
}

// Close gracefully shuts down the RabbitMQ connection and channel
func (r *RabbitMQPublisher) Close() {
	if r.channel != nil {
		r.channel.Close()
		slog.Debug("RabbitMQ channel closed")
	}
	if r.connection != nil {
		r.connection.Close()
		slog.Debug("RabbitMQ connection closed")
	}
}
