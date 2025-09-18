package services

import (
	"context"
	"net/http"
	"os"

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
func (s *AmqpService) StartAmqpConnection() *amqp.Connection {
	connection, err := amqp.Dial(s.cfg.AmqpConnection)
	if err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), map[string]interface{}{"amqp_url": s.cfg.AmqpConnection}, err)
		s.logger.LogError(context.Background(), "Failed to connect to RabbitMQ", appErr)
		os.Exit(http.StatusInternalServerError)
	}
	s.logger.Info(context.Background(), "Connected to RabbitMQ", map[string]interface{}{
		"amqp_url": s.cfg.AmqpConnection,
	})
	return connection
}

// StartChannelConnection starts the AMQP channel connection.
func (s *AmqpService) StartChannelConnection() *amqp.Channel {
	connection := s.StartAmqpConnection()
	channel, err := connection.Channel()
	if err != nil {
		appErr := errors.NewAppError(entities.ErrService, err.Error(), nil, err)
		s.logger.LogError(context.Background(), "Failed to open a channel", appErr)
		os.Exit(http.StatusInternalServerError)
	}
	s.logger.Info(context.Background(), "AMQP channel opened")
	return channel
}

// SendDataToQueue sends data to the AMQP queue.
func (s *AmqpService) SendDataToQueue(queue string, payload []byte) {
	channel := s.StartChannelConnection()

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
		os.Exit(http.StatusInternalServerError)
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
		os.Exit(http.StatusInternalServerError)
	}

	s.logger.Info(context.Background(), "Message published to queue", map[string]interface{}{
		"queue":        queue,
		"payload_size": len(payload),
	})
}

// ConsumeQueue consumes messages from the AMQP queue.
func (s *AmqpService) ConsumeQueue(queue string) <-chan amqp.Delivery {
	channel := s.StartChannelConnection()

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
		os.Exit(http.StatusInternalServerError)
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
		os.Exit(http.StatusInternalServerError)
	}

	s.logger.Info(context.Background(), "Consuming queue", map[string]interface{}{
		"queue": queue,
	})

	return msgs
}
