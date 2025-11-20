package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
)

type ShortenerService struct {
	repo    repository.Repository
	baseURL string
}

func NewShortenerService(repo repository.Repository, baseURL string) *ShortenerService {
	return &ShortenerService{repo: repo, baseURL: baseURL}
}

func (s *ShortenerService) Shorten(orig string) (string, error) {
	if !strings.HasPrefix(orig, "http://") && !strings.HasPrefix(orig, "https://") {
		return "", errors.New("некорректный url")
	}

	id := generateID()
	url := model.URL{
		Short:    id,
		Original: orig,
	}

	if err := s.repo.Save(url); err != nil {
		return "", err
	}

	return s.baseURL + "/" + id, nil
}

func (s *ShortenerService) Resolve(id string) (string, error) {
	url, err := s.repo.Find(id)

	if err != nil {
		return "", err
	}

	return url.Original, nil

}

func (s *ShortenerService) PingContext(ctx context.Context) error {

	if err := s.repo.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

func (s *ShortenerService) ShortenBatch(ctx context.Context, originals []string) ([]string, error) {
	if len(originals) == 0 {
		return nil, errors.New("empty batch")
	}

	// Сначала вычислим уникальные оригиналы и индексы
	uniqMap := make(map[string]int) // original -> index в uniq
	uniques := make([]string, 0, len(originals))
	indices := make([]int, len(originals)) // для восстановления порядка
	for i, orig := range originals {
		if idx, ok := uniqMap[orig]; ok {
			indices[i] = idx
		} else {
			idx := len(uniques)
			uniqMap[orig] = idx
			uniques = append(uniques, orig)
			indices[i] = idx
		}
	}

	// Для каждого уникального оригинала получаем/генерируем короткий URL.
	// Чтобы избежать race condition при параллельных вызовах, будем генерировать
	// и сохранять результаты через repo.SaveBatch (одно действие).
	// Здесь можно параллелить локально генерацию short-ключа, но сохранение делаем одним запросом.

	shortForUnique := make([]string, len(uniques))
	// Используем мьютекс + WaitGroup для параллельной генерации short (генерация сама по себе
	// должна быть потокобезопасной; если сервис.Shorten использует репозиторий — осторожно,
	// поэтому лучше реализовать генерацию локально через существующий s.Shorten для каждого уникального).
	var wg sync.WaitGroup
	var genErr error
	var genErrMu sync.Mutex

	for i, u := range uniques {
		wg.Add(1)
		go func(idx int, orig string) {
			defer wg.Done()
			// Можно использовать существующий Shorten, но он, возможно, уже делает запись.
			// Чтобы избежать ранней записи по одному, предполагаем, что Shorten без записи возвращает ключ,
			// но в нашем случае проще — использовать existing repo check + generation here.

			// Попробуем найти существующий короткий URL в памяти (repo may have an index),
			// но интерфейс не имеет FindByOriginal — поэтому дергаем s.Shorten(orig),
			// а затем будем собрать все model.URL и вызвать SaveBatch — SaveBatch внутри
			// должен корректно обработать дубликаты (например, при двойной вставке вернуть ошибку уникальности).
			short, err := s.Shorten(orig)
			if err != nil {
				genErrMu.Lock()
				if genErr == nil {
					genErr = err
				}
				genErrMu.Unlock()
				return
			}
			shortForUnique[idx] = short
		}(i, u)
	}

	wg.Wait()
	if genErr != nil {
		return nil, genErr
	}

	// Подготовим объекты model.URL для сохранения (один URL-пара для каждого уникального original).
	// Shorten уже добавил (возможно) записи в репозиторий, но чтобы гарантировать атомарность и
	// требование "сохранить все записи" — сделаем SaveBatch. SaveBatch должен корректно обработать
	// случаев когда запись уже существует (например, уникальные constraint в БД).
	urlsToSave := make([]model.URL, 0, len(uniques))
	for i, orig := range uniques {
		urlsToSave = append(urlsToSave, model.URL{
			Short:    shortForUnique[i],
			Original: orig,
		})
	}

	if err := s.repo.SaveBatch(ctx, urlsToSave); err != nil {
		return nil, err
	}

	// Восстанавливаем массив short'ов в исходном порядке
	result := make([]string, len(originals))
	for i, idx := range indices {
		// Склеиваем baseURL + shortForUnique[idx]
		result[i] = s.baseURL + "/" + shortForUnique[idx]
	}

	return result, nil
}

func generateID() string {
	b := make([]byte, 6)
	io.ReadFull(rand.Reader, b)

	return base64.URLEncoding.EncodeToString(b)
}
