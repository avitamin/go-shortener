package audit

import (
	"log"
	"time"
)

// Service сервис аудита, управляющий наблюдателями (Subject в паттерне Observer)
type Service struct {
	observers []Observer
}

// NewService создаёт новый сервис аудита
func NewService() *Service {
	return &Service{
		observers: make([]Observer, 0),
	}
}

// AddObserver добавляет наблюдателя
func (s *Service) AddObserver(observer Observer) {
	s.observers = append(s.observers, observer)
}

// Notify отправляет событие всем наблюдателям
func (s *Service) Notify(event Event) {
	// Отправляем асинхронно, чтобы не блокировать основной поток
	go func() {
		for _, observer := range s.observers {
			if err := observer.Notify(event); err != nil {
				log.Printf("Failed to notify observer: %v", err)
			}
		}
	}()
}

// LogShorten логирует событие создания короткой ссылки
func (s *Service) LogShorten(userID, originalURL string) {
	event := Event{
		Timestamp: time.Now().Unix(),
		Action:    ActionShorten,
		UserID:    userID,
		URL:       originalURL,
	}
	s.Notify(event)
}

// LogFollow логирует событие прохождения по короткой ссылке
func (s *Service) LogFollow(userID, originalURL string) {
	event := Event{
		Timestamp: time.Now().Unix(),
		Action:    ActionFollow,
		UserID:    userID,
		URL:       originalURL,
	}
	s.Notify(event)
}
