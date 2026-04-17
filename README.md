# Feed Docker Deployment

## Prerequisites

- Docker + Docker Compose
- Create `.env` from `.env.example` and set secure values

## Services

- Frontend: `http://localhost:3000`
- Backend API: `http://localhost:8080`
- RabbitMQ Management: `http://localhost:15672`

## Start

```bash
cp .env.example .env
# edit .env and set strong secrets first
docker compose up -d --build
```

## Stop

```bash
docker compose down
```

## Reset data

```bash
docker compose down -v
```

## Backend Test

```bash
cd feed-flow-backend
JWT_SECRET=0123456789abcdef0123456789abcdef go test ./...
```

## Frontend Build/Test

```bash
cd feed-flow-frontend
npm ci
npm run build
npm run test
```
