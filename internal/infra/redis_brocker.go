package infra

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go-ws-chat/internal/entity"
)


//go:generate go run go.uber.org/mock/mockgen -source=redis_broker.go -destination=mocks/mock_broker.go -package=mocks

// Broker — интерфейс для работы с очередью.
type Broker interface {
	Publish(ctx context.Context, msg entity.Message) error
	Consume(ctx context.Context, handler func(msg entity.Message))
	Close() error
}

type RedisBroker struct {
	client *redis.Client
	stream string
	group  string
}

func NewRedisBroker(addr, stream, group string) *RedisBroker {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	
	ctx := context.Background()
	_ = rdb.XGroupCreateMkStream(ctx, stream, group, "$").Err()

	return &RedisBroker{
		client: rdb,
		stream: stream,
		group:  group,
	}
}

// Publish использует переданный контекст для управления I/O.
func (r *RedisBroker) Publish(ctx context.Context, msg entity.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: r.stream,
		Values: map[string]interface{}{"data": data},
	}).Err()
}

// Consume использует контекст для завершения работы.
func (r *RedisBroker) Consume(ctx context.Context, onMessage func(msg entity.Message)) {
	consumerName := os.Getenv("HOSTNAME") 
	if consumerName == "" {
		consumerName = "worker-default"
	}
	
	for {
		// Проверяем контекст на отмену (Graceful Shutdown)
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled. Stopping consumer loop.", "component", "redis_consumer")
			return 
		default:
		}
		
		// Блокировка на 5 секунд, чтобы горутина могла проверить ctx.Done()
		entries, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{ 
			Group:    r.group,
			Consumer: consumerName, 
			Streams:  []string{r.stream, ">"},
			Count:    10,
			Block:    time.Second * 5, 
		}).Result()

		if err != nil && err != redis.Nil {
			slog.Error("Error reading from Redis", "error", err, "component", "redis_consumer")
			time.Sleep(time.Second) 
			continue
		}

		for _, stream := range entries {
			for _, msg := range stream.Messages {
				var message entity.Message
				msgStr := msg.Values["data"].(string)
				if err := json.Unmarshal([]byte(msgStr), &message); err != nil {
					slog.Error("Unmarshal error", "error", err, "component", "redis_consumer")
					continue
				}

				onMessage(message)
				
				r.client.XAck(ctx, r.stream, r.group, msg.ID)
			}
		}
	}
}

// Close закрывает соединение с Redis.
func (r *RedisBroker) Close() error {
    slog.Info("Closing Redis client connection.", "component", "redis_broker")
    return r.client.Close()
}