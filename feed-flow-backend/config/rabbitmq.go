package config

import (
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/rabbitmq/amqp091-go"
)

var RabbitConn *amqp091.Connection
var RabbitConsumeCh *amqp091.Channel

const VideoPublishQueue = "video_publish_queue"
const VideoPublishRetryQueue = "video_publish_retry_queue"
const VideoPublishDLX = "video_publish_dlx"
const VideoPublishDLQ = "video_publish_dlq"
const VideoPublishDLQRoutingKey = "video.publish.failed"

func InitRabbitMQ() {
	var err error
	RabbitConn, err = amqp091.Dial(getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	forceRedeclare := getEnvAsBool("RABBITMQ_FORCE_REDECLARE", false)
	if forceRedeclare {
		for _, queueName := range []string{VideoPublishQueue, VideoPublishRetryQueue, VideoPublishDLQ} {
			if delErr := deleteQueueIfExists(RabbitConn, queueName); delErr != nil {
				log.Fatalf("Failed to cleanup queue %s: %v", queueName, delErr)
			}
		}
	}

	setupCh, err := RabbitConn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %v", err)
	}
	defer setupCh.Close()

	if err = setupCh.ExchangeDeclare(
		VideoPublishDLX,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		log.Fatalf("Failed to declare dlx: %v", err)
	}

	_, err = setupCh.QueueDeclare(
		VideoPublishDLQ,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare dlq: %v", err)
	}

	if err = setupCh.QueueBind(
		VideoPublishDLQ,
		VideoPublishDLQRoutingKey,
		VideoPublishDLX,
		false,
		nil,
	); err != nil {
		log.Fatalf("Failed to bind dlq: %v", err)
	}

	_, err = setupCh.QueueDeclare(
		VideoPublishQueue,
		true,
		false,
		false,
		false,
		amqp091.Table{
			"x-dead-letter-exchange":    VideoPublishDLX,
			"x-dead-letter-routing-key": VideoPublishDLQRoutingKey,
		},
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	retryTTL := getEnvAsInt("RABBITMQ_RETRY_TTL_MS", 5000)
	_, err = setupCh.QueueDeclare(
		VideoPublishRetryQueue,
		true,
		false,
		false,
		false,
		amqp091.Table{
			"x-message-ttl":             int32(retryTTL),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": VideoPublishQueue,
		},
	)
	if err != nil {
		log.Fatalf("Failed to declare retry queue: %v", err)
	}

	RabbitConsumeCh, err = RabbitConn.Channel()
	if err != nil {
		log.Fatalf("Failed to create consumer channel: %v", err)
	}
	if err = RabbitConsumeCh.Qos(getEnvAsInt("RABBITMQ_PREFETCH", 20), 0, false); err != nil {
		log.Fatalf("Failed to set qos: %v", err)
	}

	log.Println("Connected to RabbitMQ and initialized consumer/publisher topology successfully")
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func deleteQueueIfExists(conn *amqp091.Connection, queueName string) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueueDelete(queueName, false, false, false)
	if err == nil {
		log.Printf("queue %s deleted for redeclare", queueName)
		return nil
	}

	var amqpErr *amqp091.Error
	if errors.As(err, &amqpErr) && amqpErr.Code == 404 {
		return nil
	}
	return err
}
