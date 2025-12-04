package infra

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"go-ws-chat/internal/entity"
	"log"
	"os"
	"time"
)

// Broker — интерфейс для работы с очередью
type Broker interface {
	Publish(ctx context.Context, msg entity.Message) error
	Consume(ctx context.Context, handler func(msg entity.Message))
}

type RedisBroker struct {
	client *redis.Client
	stream string
	group  string
}

func NewRedisBroker(addr, stream, group string) *RedisBroker {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	
	ctx := context.Background()
	// Создаем группу потребителей. `MkStream` создает поток, если он не существует.
	// Ошибку игнорируем, если группа уже есть.
	_ = rdb.XGroupCreateMkStream(ctx, stream, group, "$").Err()

	return &RedisBroker{
		client: rdb,
		stream: stream,
		group:  group,
	}
}

// Publish публикует сообщение в Redis Stream
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

// Consume читает сообщения из потока в бесконечном цикле
func (r *RedisBroker) Consume(ctx context.Context, onMessage func(msg entity.Message)) {
	// Имя воркера должно быть уникальным для каждого запущенного инстанса
	consumerName := os.Getenv("HOSTNAME") 
	if consumerName == "" {
		consumerName = "worker-default"
	}
	
	for {
		entries, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    r.group,
			Consumer: consumerName, 
			Streams:  []string{r.stream, ">"}, // Читаем новые (>) сообщения
			Count:    10,
			Block:    time.Second * 5, // Блокировка на 5 секунд
		}).Result()

		if err != nil && err != redis.Nil {
			log.Printf("[Redis Consumer] Error reading: %v", err)
			time.Sleep(time.Second) 
			continue
		}

		for _, stream := range entries {
			for _, msg := range stream.Messages {
				var message entity.Message
				msgStr := msg.Values["data"].(string)
				if err := json.Unmarshal([]byte(msgStr), &message); err != nil {
					log.Printf("[Redis Consumer] Unmarshal error: %v", err)
					continue
				}

				onMessage(message)
				
				// Подтверждение успешной обработки
				r.client.XAck(ctx, r.stream, r.group, msg.ID)
			}
		}
	}
}