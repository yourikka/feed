package mq

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/ranking"
)

const (
	videoPublishMaxRetry = 3
	publishTimeout       = 5 * time.Second
)

var publisherState = struct {
	mu        sync.Mutex
	conn      *amqp091.Connection
	ch        *amqp091.Channel
	confirmCh <-chan amqp091.Confirmation
}{}

// PublishVideo 发送消息：视频发布成功后异步通知
func PublishVideo(videoID uint) {
	err := PublishMessage(
		"",
		config.VideoPublishQueue,
		amqp091.Publishing{
			ContentType:  "text/plain",
			Body:         []byte(strconv.Itoa(int(videoID))),
			DeliveryMode: amqp091.Persistent,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		log.Println("mq发送消息失败:", err)
		if syncErr := processVideoPublish(int(videoID)); syncErr != nil {
			log.Printf("mq降级同步处理失败 video_id=%d err=%v", videoID, syncErr)
		}
	}
}

func ConsumerVideoMQ() {
	log.Println("mq消费者启动成功")
	consumeCh := config.GetRabbitConsumeChannel()
	if consumeCh == nil {
		log.Println("mq消费者不可用：RabbitMQ channel 未就绪")
		return
	}

	msgs, err := consumeCh.Consume(
		config.VideoPublishQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Println("mq读取消息失败:", err)
		return
	}

	for msg := range msgs {
		handleVideoMessage(msg)
	}
}

func ConsumerVideoMQWithContext(ctx context.Context) {
	log.Println("mq消费者启动成功")
	consumeCh := config.GetRabbitConsumeChannel()
	if consumeCh == nil {
		log.Println("mq消费者不可用：RabbitMQ channel 未就绪")
		return
	}

	consumerTag := fmt.Sprintf("video_publish_consumer_%d", time.Now().UnixNano())
	msgs, err := consumeCh.Consume(
		config.VideoPublishQueue,
		consumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Println("mq读取消息失败:", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("mq消费者收到停止信号，准备退出")
			if err := consumeCh.Cancel(consumerTag, false); err != nil {
				log.Printf("mq取消消费者失败: %v", err)
			}
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Println("mq消费者消息通道已关闭")
				return
			}
			handleVideoMessage(msg)
		}
	}
}

func handleVideoMessage(msg amqp091.Delivery) {
	retryCount := GetRetryCount(msg.Headers)

	videoID, err := strconv.Atoi(string(msg.Body))
	if err != nil {
		log.Printf("mq消息格式错误 body=%s: %v", string(msg.Body), err)
		handleFinalFailure(msg, retryCount, err)
		return
	}

	if err := processVideoPublish(videoID); err != nil {
		log.Printf("mq处理失败 video_id=%d retry=%d err=%v", videoID, retryCount, err)
		handleConsumeFailure(msg, retryCount, err)
		return
	}

	if err := msg.Ack(false); err != nil {
		log.Printf("mq ack失败 video_id=%d err=%v", videoID, err)
	}
}

func processVideoPublish(videoID int) error {
	if videoID <= 0 {
		return fmt.Errorf("invalid video id: %d", videoID)
	}

	log.Println("收到mq消息,处理视频发布:", videoID)
	ranking.RecordHotEvent(uint(videoID), ranking.ScorePublish)
	return nil
}

func handleConsumeFailure(msg amqp091.Delivery, retryCount int, cause error) {
	if retryCount >= videoPublishMaxRetry {
		handleFinalFailure(msg, retryCount, cause)
		return
	}

	if err := publishToRetryQueue(msg, retryCount+1, cause); err != nil {
		log.Printf("mq投递重试队列失败 retry=%d err=%v", retryCount, err)
		if nackErr := msg.Nack(false, true); nackErr != nil {
			log.Printf("mq nack失败 err=%v", nackErr)
		}
		return
	}

	if err := msg.Ack(false); err != nil {
		log.Printf("mq ack失败 err=%v", err)
	}
}

func handleFinalFailure(msg amqp091.Delivery, retryCount int, cause error) {
	if err := publishToDLQ(msg, retryCount, cause); err != nil {
		log.Printf("mq投递死信队列失败 err=%v", err)
		if nackErr := msg.Nack(false, true); nackErr != nil {
			log.Printf("mq nack失败 err=%v", nackErr)
		}
		return
	}

	if err := msg.Ack(false); err != nil {
		log.Printf("mq ack失败 err=%v", err)
	}
}

func publishToRetryQueue(msg amqp091.Delivery, retryCount int, cause error) error {
	headers := CloneHeaders(msg.Headers)
	headers["x-retry-count"] = retryCount
	headers["x-last-error"] = cause.Error()

	return PublishMessage(
		"",
		config.VideoPublishRetryQueue,
		amqp091.Publishing{
			ContentType:     msg.ContentType,
			Body:            msg.Body,
			Headers:         headers,
			DeliveryMode:    amqp091.Persistent,
			CorrelationId:   msg.CorrelationId,
			ContentEncoding: msg.ContentEncoding,
			MessageId:       msg.MessageId,
			Type:            msg.Type,
			Timestamp:       time.Now(),
		},
	)
}

func publishToDLQ(msg amqp091.Delivery, retryCount int, cause error) error {
	headers := CloneHeaders(msg.Headers)
	headers["x-retry-count"] = retryCount
	headers["x-final-error"] = cause.Error()

	return PublishMessage(
		config.VideoPublishDLX,
		config.VideoPublishDLQRoutingKey,
		amqp091.Publishing{
			ContentType:     msg.ContentType,
			Body:            msg.Body,
			Headers:         headers,
			DeliveryMode:    amqp091.Persistent,
			CorrelationId:   msg.CorrelationId,
			ContentEncoding: msg.ContentEncoding,
			MessageId:       msg.MessageId,
			Type:            msg.Type,
			Timestamp:       time.Now(),
		},
	)
}

func PublishMessage(exchange, routingKey string, publishing amqp091.Publishing) error {
	conn := config.GetRabbitConn()
	if conn == nil || conn.IsClosed() {
		return errors.New("rabbitmq unavailable")
	}

	publisherState.mu.Lock()
	defer publisherState.mu.Unlock()

	if err := ensurePublisherReady(conn); err != nil {
		return err
	}
	ch := publisherState.ch
	confirmChan := publisherState.confirmCh

	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()

	if err := ch.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false,
		false,
		publishing,
	); err != nil {
		closePublisherChannel()
		return err
	}

	select {
	case confirm, ok := <-confirmChan:
		if !ok {
			closePublisherChannel()
			return errors.New("publisher confirm channel closed")
		}
		if !confirm.Ack {
			return errors.New("publisher nack received")
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("publisher confirm timeout: %w", ctx.Err())
	}
}

func ensurePublisherReady(conn *amqp091.Connection) error {
	if publisherState.ch != nil && publisherState.conn == conn {
		return nil
	}

	closePublisherChannel()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		return err
	}

	publisherState.conn = conn
	publisherState.ch = ch
	publisherState.confirmCh = ch.NotifyPublish(make(chan amqp091.Confirmation, 256))
	return nil
}

func closePublisherChannel() {
	if publisherState.ch != nil {
		_ = publisherState.ch.Close()
	}
	publisherState.conn = nil
	publisherState.ch = nil
	publisherState.confirmCh = nil
}

func CloneHeaders(headers amqp091.Table) amqp091.Table {
	if headers == nil {
		return amqp091.Table{}
	}
	cloned := make(amqp091.Table, len(headers))
	for k, v := range headers {
		cloned[k] = v
	}
	return cloned
}

func GetRetryCount(headers amqp091.Table) int {
	if headers == nil {
		return 0
	}
	raw, ok := headers["x-retry-count"]
	if !ok {
		return 0
	}

	switch v := raw.(type) {
	case int:
		if v < 0 {
			return 0
		}
		return v
	case int8:
		if v < 0 {
			return 0
		}
		return int(v)
	case int16:
		if v < 0 {
			return 0
		}
		return int(v)
	case int32:
		if v < 0 {
			return 0
		}
		return int(v)
	case int64:
		if v < 0 {
			return 0
		}
		return int(v)
	case float64:
		if v < 0 {
			return 0
		}
		return int(v)
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 0 {
			return 0
		}
		return parsed
	default:
		return 0
	}
}
