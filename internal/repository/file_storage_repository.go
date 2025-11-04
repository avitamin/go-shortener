package repository

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/google/uuid"

	"github.com/avitamin/go-shortener/internal/model"
)

type fileStorageRepositoy struct {
	mu      sync.RWMutex
	data    map[string]string
	file    *os.File
	writer  *bufio.Writer
	encoder *json.Encoder
}

func NewFileStorageRepository(filePath string) (Repository, error) {
	// Открываем файл в режиме append, создаём если нет
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	r := &fileStorageRepositoy{
		data:   make(map[string]string),
		file:   file,
		writer: bufio.NewWriter(file),
	}

	r.encoder = json.NewEncoder(r.writer)

	if err := r.loadFromFile(); err != nil {
		panic(err)
	}

	return r, nil
}

func (r *fileStorageRepositoy) Find(id string) (model.URL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orig, ok := r.data[id]

	if !ok {
		return model.URL{}, ErrNotFound
	}

	return model.URL{ID: id, Original: orig}, nil
}

func (r *fileStorageRepositoy) Save(url model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[url.ID] = url.Original

	url.UUID = uuid.New().String()

	if err := r.encoder.Encode(url); err != nil {
		return err
	}

	if err := r.writer.Flush(); err != nil {
		return err
	}

	return r.file.Sync()

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
		r.data[u.ID] = u.Original
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
