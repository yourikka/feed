package main

import (
	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/mq"
	"github.com/yourikka/feed-flow/router"
)

func main() {
	//初始化数据库
	config.InitDB()
	//初始化redis
	config.InitRedis()
	//初始化rabbitmq
	config.InitRabbitMQ()
	//启动mq消费者
	go mq.ConsumerVideoMQ()

	//初始化路由
	r := router.SetupRouter()

	r.Static("/uploads", "./uploads")

	//启动服务
	r.Run(":8080")
}
