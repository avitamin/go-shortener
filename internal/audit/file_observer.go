package audit

import (
	"encoding/json"
	"os"
	"sync"
)

// FileObserver наблюдатель для записи событий аудита в файл
type FileObserver struct {
	filePath string
	mu       sync.Mutex
}

// NewFileObserver создает новый FileObserver для записи событий аудита в файл.
// События записываются в формате JSON (по одному событию на строку).
func NewFileObserver(filePath string) *FileObserver {
	return &FileObserver{
		filePath: filePath,
	}
}

// Notify записывает событие аудита в файл
func (f *FileObserver) Notify(event Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Открываем файл для добавления (создаём, если не существует)
	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Сериализуем событие в JSON
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Добавляем новую строку
	data = append(data, '\n')

	// Записываем в файл
	_, err = file.Write(data)
	return err
}
