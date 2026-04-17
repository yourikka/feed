# Feed Flow Backend

## Required env

- `MYSQL_DSN`
- `JWT_SECRET` (at least 32 chars)
- `FEED_HOT_WINDOW_HOURS` (optional, default `24`)

## Runtime Notes

- Redis/RabbitMQ 不可用时会降级启动，并在后台自动重连
- 发布事件优先走 MQ；MQ 不可用时会同步兜底处理
- 热榜使用 Redis 滑动窗口聚合（按小时桶）

## Run

```bash
JWT_SECRET=0123456789abcdef0123456789abcdef \
MYSQL_DSN='root:root123@tcp(127.0.0.1:3306)/douyin?charset=utf8mb4&parseTime=True&loc=Local' \
go run .
```

## Test

```bash
JWT_SECRET=0123456789abcdef0123456789abcdef go test ./...
```
