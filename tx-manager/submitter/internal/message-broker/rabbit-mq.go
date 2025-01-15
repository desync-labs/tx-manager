package message_broker

import (
	"encoding/json"
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
func (r *RabbitMQPublisher) Publish(priority int, data interface{}) error {
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
		return fmt.Errorf("invalid priority %d", priority)
	}

	// Serialize the provided data (which can be any JSON object)
	messageData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize message data: %w", err)
	}

	// Publish the message to the appropriate queue
	err = r.channel.Publish(
		"",        // Default exchange
		queueName, // The routing key (queue name)
		false,     // Mandatory
		false,     // Immediate
		amqp.Publishing{
			ContentType: "application/json", // Using application/json for content type
			Body:        messageData,        // The serialized JSON object
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
