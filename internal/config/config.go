package config

import (
	"log"
	"os"
)

// Config содержит все настройки приложения.
type Config struct {
	AppPort     string
	RedisAddr   string
	WSStream    string
	WSGroup     string
}

// LoadConfig инициализирует и возвращает конфигурацию, 
// загружая значения из переменных окружения.
func LoadConfig() *Config {
	cfg := &Config{
		AppPort:     os.Getenv("APP_PORT"),
		RedisAddr:   os.Getenv("REDIS_ADDR"),
		WSStream:    os.Getenv("WS_STREAM_NAME"),
		WSGroup:     os.Getenv("WS_GROUP_NAME"),
	}

	// Установка значений по умолчанию, если они не заданы
	if cfg.AppPort == "" {
		cfg.AppPort = "3000"
	}
	if cfg.RedisAddr == "" {
		// Используем имя сервиса из docker-compose
		cfg.RedisAddr = "redis:6379" 
	}
	if cfg.WSStream == "" {
		cfg.WSStream = "chat_stream"
	}
	if cfg.WSGroup == "" {
		cfg.WSGroup = "chat_group"
	}

	log.Printf("Loaded Config: Port=%s, Redis=%s, Stream=%s", cfg.AppPort, cfg.RedisAddr, cfg.WSStream)
	return cfg
}