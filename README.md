# Feed Docker Deployment

## Prerequisites

- Docker + Docker Compose
- Create `.env` from `.env.example` and set secure values

## Services

- Frontend: `http://localhost:3000`
- Backend API: `http://localhost:8080`
- RabbitMQ Management: `http://localhost:15672`

## Feed API Notes

- `GET /douyin/feed/?sort=latest|hot&cursor=<n>&limit=<n>`
- `sort=latest` 使用时间倒序游标分页
- `sort=hot` 使用滑动窗口热榜（Redis ZSet，`cursor` 为偏移量）
- 新版前端优先使用 `cursor_token` 翻页，后端响应 `next_token`
- `GET /douyin/feed/` 支持 `client_id`、`filter_seen=1`，可做曝光去重与已曝光过滤
- `POST /douyin/feed/event/` 用于上报 `exposure/play_start/play_progress/play_finish/pause/skip`

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
