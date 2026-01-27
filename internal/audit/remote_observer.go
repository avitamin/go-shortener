package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// RemoteObserver наблюдатель для отправки событий аудита на удалённый сервер
type RemoteObserver struct {
	url    string
	client *http.Client
}

// NewRemoteObserver создает новый RemoteObserver для отправки событий аудита на удаленный сервер.
// События отправляются HTTP POST запросом в формате JSON с таймаутом 5 секунд.
func NewRemoteObserver(url string) *RemoteObserver {
	return &RemoteObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify отправляет событие аудита на удалённый сервер методом POST
func (r *RemoteObserver) Notify(event Event) error {
	// Сериализуем событие в JSON
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Отправляем POST запрос
	resp, err := r.client.Post(r.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
