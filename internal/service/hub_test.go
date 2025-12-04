package service

import (
	"context"
	"testing"

	"go-ws-chat/internal/entity"
	mocks "go-ws-chat/internal/infra/mocks"
	"go.uber.org/mock/gomock" 
)

func TestHub_Broadcast_Success(t *testing.T) {
	// 1. Инициализация gomock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish() // Важно: проверяет, что все ожидания были выполнены

	// Создаем мок-объект для интерфейса Broker
	mockBroker := mocks.NewMockBroker(ctrl) 

	// 2. Установка Ожиданий (Expectations)
	testMsg := entity.Message{
		FromID:  "alice",
		ToID:    "bob",
		Content: "Hello Mock",
	}

	// Мы ожидаем, что метод Publish будет вызван ровно 1 раз
	// с любым контекстом (gomock.Any()) и сообщением, которое мы передали (testMsg).
	mockBroker.EXPECT().
		Publish(gomock.Any(), testMsg). // Проверяем, что аргументы совпадают
		Return(nil).                    // Симулируем успешное выполнение (Redis не упал)
		Times(1)

	// 3. Вызов тестируемого объекта (SUT - System Under Test)
	// Внедряем мок-объект в Hub (Constructor Injection)
	hub := NewHub(mockBroker, context.Background()) 
	
	// Вызываем метод
	err := hub.Broadcast(context.Background(), testMsg)

	// 4. Проверка результата
	if err != nil {
		t.Errorf("Hub.Broadcast returned an unexpected error: %v", err)
	}

	// Благодаря defer ctrl.Finish(), если Publish не был вызван (или был вызван не то количество раз),
	// тест упадет, что гораздо надежнее, чем ручная проверка булевых флагов.
}

func TestHub_Broadcast_BrokerFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBroker := mocks.NewMockBroker(ctrl)

	// Ожидаем, что Publish будет вызван и вернет ошибку (симуляция падения Redis)
	mockBroker.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(context.DeadlineExceeded). // Симулируем тайм-аут Redis
		Times(1)

	hub := NewHub(mockBroker, context.Background()) 
	
	// Вызов
	err := hub.Broadcast(context.Background(), entity.Message{})

	// Проверка: мы ожидаем, что ошибка вернется
	if err == nil {
		t.Error("Hub.Broadcast failed: expected an error due to broker failure, got nil")
	}
}