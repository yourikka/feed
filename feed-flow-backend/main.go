package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/mq"
	"github.com/yourikka/feed-flow/ranking"
	"github.com/yourikka/feed-flow/router"
	"github.com/yourikka/feed-flow/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	//初始化数据库
	config.InitDB()
	//初始化redis
	config.InitRedis()
	//初始化rabbitmq
	config.InitRabbitMQ()
	//启动mq消费者
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			mq.ConsumerVideoMQWithContext(ctx)

			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		service.StartBehaviorEventWorker(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ranking.StartHotAggRefreshWorker(ctx)
	}()

	//初始化路由
	r := router.SetupRouter()

	r.Static("/uploads", "./uploads")

	//启动服务
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server start failed: %v", err)
		}
	}()
	log.Printf("server started on :%s", port)

	<-ctx.Done()
	log.Println("shutdown signal received, exiting...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	wg.Wait()
	log.Println("server exited")
}
