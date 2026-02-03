package audit

import (
	"log"
	"sync"
	"time"
)

const defaultAuditQueueSize = 1024

// Service сервис аудита, управляющий наблюдателями (Subject в паттерне Observer)
type Service struct {
	observers []Observer
	events    chan Event
	done      chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewService создаёт новый сервис аудита
func NewService() *Service {
	svc := &Service{
		observers: make([]Observer, 0),
		events:    make(chan Event, defaultAuditQueueSize),
		done:      make(chan struct{}),
	}

	svc.wg.Add(1)
	go svc.run()

	return svc
}

// AddObserver добавляет наблюдателя
func (s *Service) AddObserver(observer Observer) {
	s.observers = append(s.observers, observer)
}

// Notify отправляет событие всем наблюдателям
func (s *Service) Notify(event Event) {
	// Очередь ограничена размером буфера; при заполнении будет блокироваться.
	// При остановке сервиса не блокируемся и не принимаем новые события.
	select {
	case s.events <- event:
	case <-s.done:
	}
}

// Close завершает работу сервиса аудита и дожидается обработки очереди.
func (s *Service) Close() {
	s.closeOnce.Do(func() {
		close(s.done)
	})
	s.wg.Wait()

	for _, observer := range s.observers {
		if closer, ok := observer.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Printf("Failed to close observer: %v", err)
			}
		}
	}
}

func (s *Service) run() {
	defer s.wg.Done()

	for {
		select {
		case event := <-s.events:
			for _, observer := range s.observers {
				if err := observer.Notify(event); err != nil {
					log.Printf("Failed to notify observer: %v", err)
				}
			}
		case <-s.done:
			// Дожидаемся обработки очереди и выходим.
			for {
				select {
				case event := <-s.events:
					for _, observer := range s.observers {
						if err := observer.Notify(event); err != nil {
							log.Printf("Failed to notify observer: %v", err)
						}
					}
				default:
					return
				}
			}
		}
	}
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
