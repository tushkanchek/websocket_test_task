.PHONY: run stop build test generate

# Запуск всего проекта (сборка + запуск Docker Compose)
run:
	docker-compose up --build

# Остановка и удаление контейнеров
stop:
	docker-compose down

# Генерация моков на основе интерфейсов
# Файл: Makefile

MOCK_TOOL = go run go.uber.org/mock/mockgen
MOCK_TARGET_PKG = go-ws-chat/internal/infra
MOCK_OUTPUT = internal/infra/mocks/mock_broker.go

# Генерация моков на основе интерфейсов (Используем Package Mode)
generate:
	# Вызываем мокген, указываем пакет и имя интерфейса, 
	# и перенаправляем вывод в файл мока.
	$(MOCK_TOOL) $(MOCK_TARGET_PKG) Broker > $(MOCK_OUTPUT)
	
# ... (остальные таргеты: test, run, stop)
test: generate
	cd internal/service && go test -v ./...