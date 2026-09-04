# Билд стадия
FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY ../go .

# Сборка
RUN go build -o main ./cmd/etl

# Финальная стадия
FROM alpine:3.20

WORKDIR /app

# Обновляем все пакеты и устанавливаем ca-certificates
RUN apk update && apk upgrade --no-cache && apk add --no-cache ca-certificates

COPY --from=builder /app/main .

EXPOSE 8080

ENTRYPOINT ["./main"]