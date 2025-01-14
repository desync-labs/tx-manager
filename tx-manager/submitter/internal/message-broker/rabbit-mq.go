package message_broker

import (
	"fmt"
	"log/slog"

	"github.com/streadway/amqp"
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
	_, err = ch.QueueDeclare("p1", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare p1 queue: %w", err)
	}

	_, err = ch.QueueDeclare("p2", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare p2 queue: %w", err)
	}

	_, err = ch.QueueDeclare("p3", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare p3 queue: %w", err)
	}

	// Return the RabbitMQPublisher instance
	return &RabbitMQPublisher{
		connection: conn,
		channel:    ch,
	}, nil
}

// Publish publishes the transaction to the appropriate RabbitMQ queue based on priority
func (r *RabbitMQPublisher) Publish(priority int, data []byte, client string) (string, error) {
	var queueName string

	// Determine the queue based on priority
	switch priority {
	case 1:
		queueName = "p1"
	case 2:
		queueName = "p2"
	case 3:
		queueName = "p3"
	default:
		return "", fmt.Errorf("invalid priority %d", priority)
	}

	// Create the message to publish (message body + routing information)
	message := fmt.Sprintf("Client: %s, Data: %s", client, string(data))

	// Publish the message to the appropriate queue
	err := r.channel.Publish(
		"",        // Default exchange
		queueName, // The routing key (queue name)
		false,     // Mandatory
		false,     // Immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to publish message: %w", err)
	}

	slog.Debug("Publishing message to message broker",
		"queue", queueName,
		"message", message,
	)

	return message, nil
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
