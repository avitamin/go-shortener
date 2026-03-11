package service_test

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/avitamin/go-shortener/internal/audit"
	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
	mock_repository "github.com/avitamin/go-shortener/internal/repository/mock"
	"github.com/avitamin/go-shortener/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	cfg *config.Config
)

func init() {
	var err error

	os.Setenv("SECRET_KEY", "secret_key")
	cfg, err = config.New(false)
	if err != nil {
		log.Fatal(err)
	}
}

func TestShorterenerService(t *testing.T) {
	tests := []struct {
		name               string
		baseURL            string
		originalURL        string
		invalidOriginalURL string
		shortURL           string
		wrongShortURL      string
		testShorten        bool
		expectShortenError bool
		testResolve        bool
		expectResolveError bool
		wantResolveError   error
		wantResolvedURL    string
		wantEmptyShortURL  bool
	}{
		{
			name:               "shorten success",
			testShorten:        true,
			testResolve:        false,
			baseURL:            "http://localhost:8080",
			originalURL:        "https://yandex.ru",
			shortURL:           "http://localhost:8080/short",
			expectShortenError: false,
			expectResolveError: false,
			wantEmptyShortURL:  false,
		},
		{
			name:               "shorten with invalid URL",
			testResolve:        true,
			testShorten:        false,
			baseURL:            "http://localhost:8080",
			invalidOriginalURL: "yandex.ru",
			shortURL:           "http://localhost:8080/short",
			expectShortenError: true,
			expectResolveError: false,
			wantEmptyShortURL:  true,
		},
		{
			name:               "resolve success",
			testShorten:        false,
			testResolve:        true,
			baseURL:            "http://localhost:8080",
			originalURL:        "https://yandex.ru",
			shortURL:           "short",
			expectShortenError: false,
			expectResolveError: false,
			wantResolvedURL:    "https://yandex.ru",
			wantEmptyShortURL:  false,
		},
		{
			name:               "resolve not found",
			testShorten:        false,
			testResolve:        true,
			baseURL:            "http://localhost:8080",
			originalURL:        "https://yandex.ru",
			wrongShortURL:      "wrong url",
			expectShortenError: false,
			expectResolveError: true,
			wantResolveError:   repository.ErrNotFound,
			wantEmptyShortURL:  false,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_repository.NewMockRepository(ctrl)
	defer repo.Close()

	repo.EXPECT().Close().Return(nil).AnyTimes()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.BaseURL = tt.baseURL
			svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))

			// Добавляем userId в контекст
			ctx := context.WithValue(context.Background(), model.ContextUserID, "test-user-id")
			defer ctx.Done()

			if tt.testShorten {
				if tt.invalidOriginalURL != "" {
					repo.EXPECT().Save(ctx, gomock.Any()).Return(errors.New("некорректный url")).AnyTimes()
				} else {
					repo.EXPECT().Save(ctx, gomock.Any()).Return(nil).AnyTimes()
				}

				shortURL, err := svc.Shorten(ctx, tt.originalURL)
				if tt.expectShortenError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				if tt.wantEmptyShortURL {
					assert.Empty(t, shortURL)
				} else {
					assert.NotEmpty(t, shortURL)

					assert.Contains(t, shortURL, tt.baseURL)
				}
			}

			if tt.testResolve {
				var shortURL string
				if tt.wrongShortURL != "" {
					shortURL = tt.wrongShortURL
					repo.EXPECT().Find(tt.wrongShortURL).Return(model.URL{}, repository.ErrNotFound).AnyTimes()
				} else {
					shortURL = tt.shortURL
				}

				if tt.shortURL != "" {
					repo.EXPECT().Find(tt.shortURL).Return(model.URL{Short: tt.shortURL, Original: tt.originalURL}, nil).AnyTimes()
				}

				result, err := svc.Resolve(shortURL)

				if tt.expectResolveError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				if tt.wantResolveError != nil {
					assert.ErrorIs(t, err, tt.wantResolveError)
				}

				if tt.wantResolvedURL != "" {
					assert.Equal(t, tt.wantResolvedURL, result)
				}
			}
		})
	}

}

func TestShortenerServiceShutdownFlushesDeleteQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_repository.NewMockRepository(ctrl)
	repo.EXPECT().DeleteUserURLs(gomock.Any(), "test-user-id", []string{"short1", "short2"}).Return(nil).Times(1)

	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))

	ctx := context.WithValue(context.Background(), model.ContextUserID, "test-user-id")
	err := svc.DeleteUserURLs(ctx, []string{"short1", "short2"})
	assert.NoError(t, err)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = svc.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}

func TestShortenerServiceDeleteUserURLsReturnsErrorAfterShutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_repository.NewMockRepository(ctrl)
	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := svc.Shutdown(shutdownCtx)
	assert.NoError(t, err)

	ctx := context.WithValue(context.Background(), model.ContextUserID, "test-user-id")
	err = svc.DeleteUserURLs(ctx, []string{"short1"})
	assert.ErrorIs(t, err, service.ErrServiceShuttingDown)
}

func TestShortenerServiceShutdownRetriesDeleteQueueUntilSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_repository.NewMockRepository(ctrl)
	gomock.InOrder(
		repo.EXPECT().DeleteUserURLs(gomock.Any(), "test-user-id", []string{"short1"}).Return(errors.New("temporary delete error")),
		repo.EXPECT().DeleteUserURLs(gomock.Any(), "test-user-id", []string{"short1"}).Return(nil),
	)

	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.WithValue(context.Background(), model.ContextUserID, "test-user-id")
	err := svc.DeleteUserURLs(ctx, []string{"short1"})
	assert.NoError(t, err)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	startedAt := time.Now()
	err = svc.Shutdown(shutdownCtx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, time.Since(startedAt), 90*time.Millisecond)
}

func TestShortenerServiceShutdownStopsRetriesWhenContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_repository.NewMockRepository(ctrl)
	repo.EXPECT().DeleteUserURLs(gomock.Any(), "test-user-id", []string{"short1"}).Return(errors.New("permanent delete error")).AnyTimes()

	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.WithValue(context.Background(), model.ContextUserID, "test-user-id")
	err := svc.DeleteUserURLs(ctx, []string{"short1"})
	assert.NoError(t, err)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = svc.Shutdown(shutdownCtx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
