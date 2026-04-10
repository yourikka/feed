package config

import (
	"log"

	"github.com/rabbitmq/amqp091-go"
)

var RabbitConn *amqp091.Connection
var RabbitCh *amqp091.Channel

const VideoPublishQueue = "video_publish_queue"

func InitRabbitMQ() {
	var err error
	RabbitConn, err = amqp091.Dial(getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	RabbitCh, err = RabbitConn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %v", err)
	}

	_, err = RabbitCh.QueueDeclare(
		VideoPublishQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	log.Println("Connected to RabbitMQ and declared queue successfully")
}
