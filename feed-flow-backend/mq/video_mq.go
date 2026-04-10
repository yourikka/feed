package mq

import (
	"log"
	"strconv"

	"github.com/rabbitmq/amqp091-go"
	"github.com/yourikka/feed-flow/config"
)

// 队列名称
const VideoPublishStream = "video_publish_stream"

// PublishVideo 发送消息：视频发布成功后异步通知
func PublishVideo(videoId uint) {
	err := config.RabbitCh.Publish(
		"", //默认交换机
		config.VideoPublishQueue,
		false,
		false,
		amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(strconv.Itoa(int(videoId))),
		},
	)
	if err != nil {
		log.Println("mq发送消息失败:", err)
	}
}

func ConsumerVideoMQ() {
	log.Println("mq消费者启动成功")

	msgs, err := config.RabbitCh.Consume(
		config.VideoPublishQueue,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Println("mq读取消息失败:", err)
	}

	//处理消息
	for msg := range msgs {
		videoID, _ := strconv.Atoi(string(msg.Body))
		log.Println("收到mq消息,处理视频发布:", videoID)

		// ========== 这里写你的异步业务逻辑 ==========
		// 1. 视频审核
		// 2. 热度统计
		// 3. 粉丝推送
		// 4. 搜索索引构建
	}
}
