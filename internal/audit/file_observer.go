package audit

import (
	"encoding/json"
	"os"
	"sync"
)

// FileObserver наблюдатель для записи событий аудита в файл
type FileObserver struct {
	filePath string
	file     *os.File
	closed   bool
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

	if f.closed {
		return os.ErrClosed
	}

	if f.file == nil {
		// Открываем файл для добавления (создаём, если не существует)
		file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		f.file = file
	}

	// Сериализуем событие в JSON
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Добавляем новую строку
	data = append(data, '\n')

	// Записываем в файл
	_, err = f.file.Write(data)
	return err
}

// Close закрывает файл наблюдателя.
func (f *FileObserver) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return nil
	}
	f.closed = true

	if f.file == nil {
		return nil
	}

	err := f.file.Close()
	f.file = nil
	return err
}
