# Feed Flow Backend

## Required env

- `MYSQL_DSN`
- `JWT_SECRET` (at least 32 chars)

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
