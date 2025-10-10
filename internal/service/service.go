package service

import (
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
	baseUrl string
}

func NewShortenerService(repo repository.Repository, baseUrl string) *ShortenerService {
	return &ShortenerService{repo: repo, baseUrl: baseUrl}
}

func (s *ShortenerService) Shorten(orig string) (string, error) {
	if !strings.HasPrefix(orig, "http://") && !strings.HasPrefix(orig, "https://") {
		return "", errors.New("Некорректный url")
	}

	id := generateID()
	url := model.Url{
		ID:       id,
		Original: orig,
	}

	if err := s.repo.Save(url); err != nil {
		return "", err
	}

	return s.baseUrl + "/" + id, nil
}

func (s *ShortenerService) Resolve(id string) (string, error) {
	url, err := s.repo.Find(id)

	if err != nil {
		return "", err
	}

	return url.Original, nil

}

func generateID() string {
	b := make([]byte, 6)
	_, _ = io.ReadFull(rand.Reader, b)

	return base64.URLEncoding.EncodeToString(b)
}
