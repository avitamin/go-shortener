package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"

	"github.com/avitamin/go-shortener/internal/model"
)

type fileStorageRepositoy struct {
	mu      sync.Mutex
	storage *inMemoryStorage
	file    *os.File
	writer  *bufio.Writer
	encoder *json.Encoder
}

func NewFileStorageRepository(filePath string) (Repository, error) {
	// Открываем файл в режиме append, создаём если нет
	file, err := os.OpenFile(filepath.Clean(filePath), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	r := &fileStorageRepositoy{
		storage: NewInMemoryStorage(),
		file:    file,
		writer:  bufio.NewWriter(file),
	}

	r.encoder = json.NewEncoder(r.writer)

	if err := r.loadFromFile(); err != nil {
		return r, err
	}

	return r, nil
}

func (r *fileStorageRepositoy) Find(id string) (model.URL, error) {
	return r.storage.Find(id)
}

func (r *fileStorageRepositoy) GetShort(orig string) (short string, ok bool) {
	return r.storage.GetShort(orig)
}

func (r *fileStorageRepositoy) Save(url model.URL) error {
	if err := r.storage.Save(url); err != nil {
		return err
	}

	err := r.writeURL(url)
	if err != nil {
		return err
	}

	if err := r.writer.Flush(); err != nil {
		return err
	}

	return r.file.Sync()

}

func (r *fileStorageRepositoy) writeURL(url model.URL) error {
	url.UUID = uuid.NewString()

	if err := r.encoder.Encode(url); err != nil {
		return err
	}

	return nil
}

func (r *fileStorageRepositoy) SaveBatch(ctx context.Context, urls []model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Сначала сохраняем в in-memory (без race)
	if err := r.storage.SaveBatchNoLock(ctx, urls); err != nil {
		return err
	}

	// Теперь пишем в файл все URL как одну транзакцию
	for _, u := range urls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			u.UUID = uuid.New().String()

			if err := r.encoder.Encode(u); err != nil {
				return err
			}
		}
	}

	// Сбрасываем буфер
	if err := r.writer.Flush(); err != nil {
		return err
	}

	// Фиксируем транзакцию в файле
	if err := r.file.Sync(); err != nil {
		return err
	}

	return nil
}

func (r *fileStorageRepositoy) PingContext(ctx context.Context) error {
	return errors.New("db not configured")
}

func (r *fileStorageRepositoy) loadFromFile() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dec := json.NewDecoder(r.file)

	for {
		var u model.URL
		if err := dec.Decode(&u); err != nil {
			if errors.Is(err, os.ErrInvalid) {
				// Битая строка в файле, пропускаем
				break
			}
			if errors.Is(err, io.EOF) {
				// Пустой файл или достигнут конец файла
				break
			}
			// пропускаем битые строки
			continue
		}
		r.storage.Save(u)
	}
	return nil
}

func (r *fileStorageRepositoy) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.writer.Flush()

	if r.file != nil {
		return r.file.Close()
	}

	return nil
}
