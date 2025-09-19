package services

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	amqp "github.com/rabbitmq/amqp091-go"
)

// AmqpService provides AMQP messaging capabilities.
type AmqpService struct {
	logger logger.Logger
	cfg    *config.AppConfig
}

// NewAmqpService creates a new AmqpService instance.
func NewAmqpService(logger logger.Logger, cfg *config.AppConfig) *AmqpService {
	return &AmqpService{logger: logger, cfg: cfg}
}

// StartAmqpConnection starts the AMQP connection.
func (s *AmqpService) StartAmqpConnection() (*amqp.Connection, error) {
	connection, err := amqp.Dial(s.cfg.AmqpConnection)
	if err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{"amqp_url": s.cfg.AmqpConnection}, err)
		s.logger.LogError(context.Background(), "Failed to connect to RabbitMQ", appErr)
		return nil, err
	}
	s.logger.Info(context.Background(), "Connected to RabbitMQ", map[string]interface{}{
		"amqp_url": s.cfg.AmqpConnection,
	})
	return connection, nil
}

// StartChannelConnection starts the AMQP channel connection.
func (s *AmqpService) StartChannelConnection() (*amqp.Channel, error) {
	connection, err := s.StartAmqpConnection()
	if err != nil {
		return nil, err
	}

	channel, err := connection.Channel()
	if err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), nil, err)
		s.logger.LogError(context.Background(), "Failed to open a channel", appErr)
		return nil, err
	}
	s.logger.Info(context.Background(), "AMQP channel opened")
	return channel, nil
}

// SendDataToQueue sends data to the AMQP queue.
func (s *AmqpService) SendDataToQueue(queue string, payload []byte) error {
	channel, err := s.StartChannelConnection()
	if err != nil {
		s.logger.Error(context.Background(), "AMQP not available, skipping queue operation", map[string]interface{}{
			"queue": queue,
			"error": err.Error(),
		})
		return err
	}

	q, internalError := channel.QueueDeclare(
		queue, // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	if internalError != nil {
		appErr := errors.NewAppError(entities.ErrService, internalError.Error(), map[string]interface{}{"queue": queue}, internalError)
		s.logger.LogError(context.Background(), "Failed to declare queue", appErr)
		return internalError
	}

	internalError = channel.PublishWithContext(context.Background(),
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payload,
		})

	if internalError != nil {
		appErr := errors.NewAppError(entities.ErrService, internalError.Error(), map[string]interface{}{"queue": queue}, internalError)
		s.logger.LogError(context.Background(), "Failed to publish message", appErr)
		return internalError
	}

	s.logger.Info(context.Background(), "Message published to queue", map[string]interface{}{
		"queue":        queue,
		"payload_size": len(payload),
	})
	return nil
}

// ConsumeQueue consumes messages from the AMQP queue.
func (s *AmqpService) ConsumeQueue(queue string) (<-chan amqp.Delivery, error) {
	channel, err := s.StartChannelConnection()
	if err != nil {
		s.logger.Error(context.Background(), "AMQP not available, cannot consume queue", map[string]interface{}{
			"queue": queue,
			"error": err.Error(),
		})
		return nil, err
	}

	q, internalError := channel.QueueDeclare(
		queue, // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	if internalError != nil {
		appErr := errors.NewAppError(entities.ErrService, internalError.Error(), map[string]interface{}{"queue": queue}, internalError)
		s.logger.LogError(context.Background(), "Failed to declare queue for consume", appErr)
		return nil, internalError
	}

	msgs, internalError := channel.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)

	if internalError != nil {
		appErr := errors.NewAppError(entities.ErrService, internalError.Error(), map[string]interface{}{"queue": queue}, internalError)
		s.logger.LogError(context.Background(), "Failed to start consuming queue", appErr)
		return nil, internalError
	}

	s.logger.Info(context.Background(), "Consuming queue", map[string]interface{}{
		"queue": queue,
	})

	return msgs, nil
}
