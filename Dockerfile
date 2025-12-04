FROM golang:1.22-alpine AS builder
WORKDIR /app

# --- НОВЫЕ СТРОКИ ДЛЯ УСТРАНЕНИЯ ОШИБОК СЕТИ ---
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org
# -----------------------------------------------

COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Сборка приложения
RUN go build -o app ./cmd/api/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/app .
EXPOSE 3000
CMD ["./app"]