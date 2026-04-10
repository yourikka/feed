# Feed Docker Deployment

## Services

- Frontend: `http://localhost:3000`
- Backend API: `http://localhost:8080`
- RabbitMQ Management: `http://localhost:15672`

## Start

```bash
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
