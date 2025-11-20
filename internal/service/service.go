package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"

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
	err := s.validateOriginalURL(orig)
	if err != nil {
		return "", err
	}

	short, url := s.createModel(orig)

	if err := s.repo.Save(url); err != nil {
		return "", err
	}

	return s.baseURL + "/" + short, nil
}

func (s *ShortenerService) createModel(orig string) (string, model.URL) {
	short := generateID()

	model := model.URL{
		Short:    short,
		Original: orig,
	}

	return short, model
}

func (s *ShortenerService) validateOriginalURL(orig string) error {

	if !strings.HasPrefix(orig, "http://") && !strings.HasPrefix(orig, "https://") {
		return errors.New("некорректный url")
	}

	return nil
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

func (s *ShortenerService) GetShort(orig string) (string, bool) {
	return s.repo.GetShort(orig)
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
	shortForUnique := make([]string, len(uniques))

	urlsToSave := make([]model.URL, 0, len(uniques))
	for i, orig := range uniques {
		short, ok := s.repo.GetShort(orig)
		if ok {
			shortForUnique[i] = short
			continue
		}

		short, model := s.createModel(orig)
		shortForUnique[i] = short

		urlsToSave = append(urlsToSave, model)
	}

	if err := s.repo.SaveBatch(ctx, urlsToSave); err != nil {
		return nil, err
	}

	// Восстанавливаем массив short'ов в исходном порядке
	result := make([]string, len(originals))
	for i, idx := range indices {

		result[i] = s.baseURL + "/" + shortForUnique[idx]
	}

	return result, nil
}

func generateID() string {
	b := make([]byte, 6)
	io.ReadFull(rand.Reader, b)

	return base64.URLEncoding.EncodeToString(b)
}
