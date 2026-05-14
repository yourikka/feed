# Feed Docker Deployment

## Prerequisites

- Docker + Docker Compose
- Create `.env` from `.env.example` and set secure values

## Services

- Frontend: `http://localhost:3000`
- Backend API: `http://localhost:8080`
- RabbitMQ Management: `http://localhost:15672`

## Feed API Notes

- `GET /douyin/feed/?sort=latest|hot|follow&cursor=<n>&limit=<n>`
- `GET /douyin/feed/ids/?sort=latest|hot|follow&cursor_token=<token>&limit=<n>` 先获取视频 ID 列表
- `GET /douyin/feed/details/?video_ids=1,2,3` 按 ID 批量取视频详情
- `sort=latest` 使用时间倒序游标分页
- `sort=hot` 使用滑动窗口热榜（Redis ZSet，`cursor` 为偏移量）
- 新版前端优先使用 `cursor_token` 翻页，后端响应 `next_token`
- `GET /douyin/feed/` 支持 `client_id`、`filter_seen=1`，可做曝光去重与已曝光过滤
- 前端预加载改成“先拉 ID，再按 ID 从 Redis/接口回填详情”，避免直接预取完整视频列表
- `POST /douyin/feed/event/` 用于上报 `exposure/play_start/play_progress/play_finish/pause/skip`
- 播放行为事件现在走 RabbitMQ 异步消费，不再使用 Redis 简单队列
- 热榜聚合支持后台预热刷新，减少请求时实时聚合开销
- 用户态视频卡片支持 viewer 维度短缓存，缓存点赞/收藏/关注状态
- 关注流支持推拉结合：普通作者写扩散到 inbox，大V使用拉模式实时聚合

## Start

```bash
cp .env.example .env
# edit .env and set strong secrets first
# keep MYSQL_DSN password same as MYSQL_ROOT_PASSWORD
docker compose up -d --build
docker compose ps
```

## Stop

```bash
docker compose down
```

## Reset data

```bash
docker compose down -v
```

## Smoke Test

```bash
./scripts/smoke.sh
```

## Useful Commands

```bash
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f mysql redis rabbitmq
```

## K6 Load Test

```bash
# install k6 first, then run:
k6 run -e BASE_URL=http://localhost:8080 scripts/k6/feed.js

# extreme mode (single latest-feed endpoint, no sleep, ramping arrival rate):
k6 run -e TEST_MODE=extreme -e BASE_URL=http://localhost:8080 scripts/k6/feed.js

# online-like mixed traffic mode (auth + latest/hot/comment/like, includes steady 3k RPS stage):
k6 run -e BASE_URL=http://localhost:8080 scripts/k6/feed_online_mix.js

# optional: reuse existing token + tune setup token pool size
k6 run -e BASE_URL=http://localhost:8080 -e TOKEN="<your_token>" scripts/k6/feed_online_mix.js
k6 run -e BASE_URL=http://localhost:8080 -e TOKEN_POOL_SIZE=50 scripts/k6/feed_online_mix.js
```

## Backend Test

```bash
cd feed-flow-backend
export GOCACHE=/home/rikka/.marscode/.gocache
export GOMODCACHE=/home/rikka/.marscode/.gomodcache
JWT_SECRET=0123456789abcdef0123456789abcdef go test ./...
```

## Frontend Build/Test

```bash
cd feed-flow-frontend
npm ci
npm run build
npm run test
```

## CI/CD

- GitHub Actions CI 会自动执行：
  - 后端 `go test ./...`
  - 前端 `npm run build` 和 `npm run test`
  - 前后端 Docker 镜像构建校验
  - `docker compose` 启动完整依赖并执行 `./scripts/smoke.sh`
- 当代码 push 到 `main` 分支后，流水线还会自动把镜像发布到 GHCR：
  - `ghcr.io/<owner>/feed-backend`
  - `ghcr.io/<owner>/feed-frontend`
