# Этап сборки
FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o app ./cmd/api/main.go

# Финальный образ
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/app .
EXPOSE 3000
CMD ["./app"]