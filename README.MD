# Go WebSocket Chat с Redis Streams

## Описание проекта
Это реализация WebSocket-сервера на Go (Fiber) с использованием **Redis Streams** в качестве асинхронной очереди сообщений. Архитектура построена по принципам Clean Architecture с четким разделением на слои (Handler, Service, Infrastructure) и использованием **интерфейсов** (`Broker`) для инверсии зависимостей.

## Инструкция по запуску (Docker Compose)

1. **Клонировать репозиторий** (или создать структуру файлов).
2. **Запустить Docker Compose:**
   ```bash
   docker-compose up --build