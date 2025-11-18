package service_test

import (
	"testing"

	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
	"github.com/stretchr/testify/assert"
)

// mock repository for isolation
type mockRepo struct {
	data map[string]string
}

func newMockRepo() *mockRepo {
	return &mockRepo{data: make(map[string]string)}
}

func (m *mockRepo) Save(url model.URL) error {
	m.data[url.Short] = url.Original
	return nil
}

func (m *mockRepo) Find(id string) (model.URL, error) {
	val, ok := m.data[id]
	if !ok {
		return model.URL{}, repository.ErrNotFound
	}
	return model.URL{Short: id, Original: val}, nil
}

func (m *mockRepo) Close() error {
	return nil
}

func TestShorten_Success(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewShortenerService(repo, "http://localhost:8080")

	shortURL, err := svc.Shorten("https://yandex.ru")
	assert.NoError(t, err)
	assert.Contains(t, shortURL, "http://localhost:8080/")
	assert.Len(t, repo.data, 1)
}

func TestShorten_InvalidURL(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewShortenerService(repo, "http://localhost:8080")

	shortURL, err := svc.Shorten("yandex.ru")
	assert.Error(t, err)
	assert.Empty(t, shortURL)
}

func TestResolve_Success(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewShortenerService(repo, "http://localhost:8080")

	url := model.URL{Short: "abc123", Original: "https://ya.ru"}
	_ = repo.Save(url)

	result, err := svc.Resolve("abc123")
	assert.NoError(t, err)
	assert.Equal(t, "https://ya.ru", result)
}

func TestResolve_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewShortenerService(repo, "http://localhost:8080")

	_, err := svc.Resolve("notfound")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}
