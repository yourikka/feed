# Feed Flow Backend

## Required env

- `MYSQL_DSN`
- `JWT_SECRET` (at least 32 chars)
- `FEED_HOT_WINDOW_HOURS` (optional, default `24`)
- `FEED_FOLLOW_PUSH_MAX_FANS` (optional, default `2000`，关注流推拉阈值)
- `DB_AUTO_MIGRATE` (optional, `true` 时自动创建 `video_behavior_events` 等表)
- `SNAPSHOT_SECRET` (optional, 不配时回退到 `JWT_SECRET`，用于热榜快照 token 签名)

## Runtime Notes

- Redis/RabbitMQ 不可用时会降级启动，并在后台自动重连
- 发布事件优先走 MQ；MQ 不可用时会同步兜底处理
- 热榜使用 Redis 滑动窗口聚合（按小时桶）
- Feed 支持基于 `client_id/user_id` 的曝光去重和最近曝光过滤
- 热榜分页支持带签名的快照 token，避免动态榜单 offset 翻页重复/漏数
- 热榜聚合 key 使用 Redis 短缓存，并由后台 worker 周期预热
- 视频详情优先从 Redis 读取，前端预加载改成先拉视频 ID 再批量取详情
- 播放行为事件支持 `exposure/play_start/play_progress/play_finish/pause/skip`
- 行为事件优先走 RabbitMQ 异步消费，失败时再同步落库
- Feed 卡片新增 viewer 维度缓存，缓存用户交互状态后的完整对象
- 关注流支持推拉结合：普通作者写扩散到 `follow_feed_inboxes`，大V走拉模式

## Run

```bash
JWT_SECRET=0123456789abcdef0123456789abcdef \
MYSQL_DSN='root:root123@tcp(127.0.0.1:3306)/douyin?charset=utf8mb4&parseTime=True&loc=Local' \
go run .
```

## Test

```bash
export GOCACHE=/home/rikka/.marscode/.gocache
export GOMODCACHE=/home/rikka/.marscode/.gomodcache
JWT_SECRET=0123456789abcdef0123456789abcdef go test ./...
```
