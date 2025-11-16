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
	db      Pinger
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
		ID:       id,
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

type Pinger interface {
	PingContext(ctx context.Context) error
}

func (s *ShortenerService) PingContext(ctx context.Context) error {
	if s.db == nil {
		panic("не задано свойство db")
	}

	if err := s.db.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

func (s *ShortenerService) AttachDB(db Pinger) error {
	s.db = db

	return nil
}

func generateID() string {
	b := make([]byte, 6)
	io.ReadFull(rand.Reader, b)

	return base64.URLEncoding.EncodeToString(b)
}
