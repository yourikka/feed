package config

import (
	"errors"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

var (
	rabbitMu        sync.RWMutex
	RabbitConn      *amqp091.Connection
	RabbitConsumeCh *amqp091.Channel
)

const (
	VideoPublishQueue          = "video_publish_queue"
	VideoPublishRetryQueue     = "video_publish_retry_queue"
	VideoPublishDLX            = "video_publish_dlx"
	VideoPublishDLQ            = "video_publish_dlq"
	VideoPublishDLQRoutingKey  = "video.publish.failed"
	BehaviorEventQueue         = "behavior_event_queue"
	BehaviorEventRetryQueue    = "behavior_event_retry_queue"
	BehaviorEventDLX           = "behavior_event_dlx"
	BehaviorEventDLQ           = "behavior_event_dlq"
	BehaviorEventDLQRoutingKey = "behavior.event.failed"
	rabbitReconnectInterval    = 5 * time.Second
)

func InitRabbitMQ() {
	if err := connectAndSetupRabbit(); err != nil {
		log.Printf("RabbitMQ unavailable, enter degraded mode: %v", err)
	}
	go keepRabbitAlive()
}

func keepRabbitAlive() {
	ticker := time.NewTicker(rabbitReconnectInterval)
	defer ticker.Stop()

	for range ticker.C {
		if IsRabbitMQReady() {
			continue
		}
		if err := connectAndSetupRabbit(); err != nil {
			log.Printf("RabbitMQ reconnect failed: %v", err)
			continue
		}
		log.Println("RabbitMQ self-heal succeeded")
	}
}

func connectAndSetupRabbit() error {
	conn, err := amqp091.Dial(getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"))
	if err != nil {
		return err
	}

	forceRedeclare := getEnvAsBool("RABBITMQ_FORCE_REDECLARE", false)
	if forceRedeclare {
		for _, queueName := range []string{
			VideoPublishQueue, VideoPublishRetryQueue, VideoPublishDLQ,
			BehaviorEventQueue, BehaviorEventRetryQueue, BehaviorEventDLQ,
		} {
			if delErr := deleteQueueIfExists(conn, queueName); delErr != nil {
				_ = conn.Close()
				return delErr
			}
		}
	}

	setupCh, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err = setupTopology(setupCh); err != nil {
		_ = setupCh.Close()
		_ = conn.Close()
		return err
	}
	_ = setupCh.Close()

	consumeCh, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err = consumeCh.Qos(getEnvAsInt("RABBITMQ_PREFETCH", 20), 0, false); err != nil {
		_ = consumeCh.Close()
		_ = conn.Close()
		return err
	}

	setRabbitState(conn, consumeCh)
	log.Println("Connected to RabbitMQ and initialized consumer/publisher topology successfully")
	return nil
}

func setupTopology(ch *amqp091.Channel) error {
	if err := ch.ExchangeDeclare(
		VideoPublishDLX,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(
		VideoPublishDLQ,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	if err := ch.QueueBind(
		VideoPublishDLQ,
		VideoPublishDLQRoutingKey,
		VideoPublishDLX,
		false,
		nil,
	); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(
		VideoPublishQueue,
		true,
		false,
		false,
		false,
		amqp091.Table{
			"x-dead-letter-exchange":    VideoPublishDLX,
			"x-dead-letter-routing-key": VideoPublishDLQRoutingKey,
		},
	); err != nil {
		return err
	}

	retryTTL := getEnvAsInt("RABBITMQ_RETRY_TTL_MS", 5000)
	if _, err := ch.QueueDeclare(
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
	); err != nil {
		return err
	}

	if err := ch.ExchangeDeclare(
		BehaviorEventDLX,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(
		BehaviorEventDLQ,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	if err := ch.QueueBind(
		BehaviorEventDLQ,
		BehaviorEventDLQRoutingKey,
		BehaviorEventDLX,
		false,
		nil,
	); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(
		BehaviorEventQueue,
		true,
		false,
		false,
		false,
		amqp091.Table{
			"x-dead-letter-exchange":    BehaviorEventDLX,
			"x-dead-letter-routing-key": BehaviorEventDLQRoutingKey,
		},
	); err != nil {
		return err
	}

	retryTTL = getEnvAsInt("RABBITMQ_RETRY_TTL_MS", 5000)
	if _, err := ch.QueueDeclare(
		BehaviorEventRetryQueue,
		true,
		false,
		false,
		false,
		amqp091.Table{
			"x-message-ttl":             int32(retryTTL),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": BehaviorEventQueue,
		},
	); err != nil {
		return err
	}

	return nil
}

func setRabbitState(conn *amqp091.Connection, consumeCh *amqp091.Channel) {
	rabbitMu.Lock()
	oldConn := RabbitConn
	oldCh := RabbitConsumeCh
	RabbitConn = conn
	RabbitConsumeCh = consumeCh
	rabbitMu.Unlock()

	if oldCh != nil && oldCh != consumeCh {
		_ = oldCh.Close()
	}
	if oldConn != nil && oldConn != conn {
		_ = oldConn.Close()
	}
}

func GetRabbitConn() *amqp091.Connection {
	rabbitMu.RLock()
	defer rabbitMu.RUnlock()
	return RabbitConn
}

func GetRabbitConsumeChannel() *amqp091.Channel {
	rabbitMu.RLock()
	defer rabbitMu.RUnlock()
	return RabbitConsumeCh
}

func IsRabbitMQReady() bool {
	conn := GetRabbitConn()
	ch := GetRabbitConsumeChannel()
	if conn == nil || ch == nil {
		return false
	}
	return !conn.IsClosed()
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
