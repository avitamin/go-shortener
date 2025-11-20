package service_test

import (
	"errors"
	"testing"

	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
	mock_repository "github.com/avitamin/go-shortener/internal/repository/mock"
	"github.com/avitamin/go-shortener/internal/service"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestShorterenerService(t *testing.T) {
	tests := []struct {
		name               string
		baseURL            string
		originalURL        string
		invalidOriginalURL string
		shortURL           string
		wrongShortURL      string
		testShorten        bool
		wantShortenError   bool
		testResolve        bool
		wantResolveError   bool
		wantResolveErrorIs error
		wantResolvedUrl    string
		wantEmptyShortUrl  bool
	}{
		{
			name:              "shorten success",
			testShorten:       true,
			testResolve:       false,
			baseURL:           "http://localhost:8080",
			originalURL:       "https://yandex.ru",
			shortURL:          "http://localhost:8080/short",
			wantShortenError:  false,
			wantResolveError:  false,
			wantEmptyShortUrl: false,
		},
		{
			name:               "shorten with invalid URL",
			testResolve:        true,
			testShorten:        false,
			baseURL:            "http://localhost:8080",
			invalidOriginalURL: "yandex.ru",
			shortURL:           "http://localhost:8080/short",
			wantShortenError:   true,
			wantResolveError:   false,
			wantEmptyShortUrl:  true,
		},
		{
			name:              "resolve success",
			testShorten:       false,
			testResolve:       true,
			baseURL:           "http://localhost:8080",
			originalURL:       "https://yandex.ru",
			shortURL:          "short",
			wantShortenError:  false,
			wantResolveError:  false,
			wantResolvedUrl:   "https://yandex.ru",
			wantEmptyShortUrl: false,
		},
		{
			name:               "resolve not found",
			testShorten:        false,
			testResolve:        true,
			baseURL:            "http://localhost:8080",
			originalURL:        "https://yandex.ru",
			wrongShortURL:      "wrong url",
			wantShortenError:   false,
			wantResolveError:   true,
			wantResolveErrorIs: repository.ErrNotFound,
			wantEmptyShortUrl:  false,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_repository.NewMockRepository(ctrl)
	defer repo.Close()

	repo.EXPECT().Close().Return(nil).AnyTimes()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewShortenerService(repo, tt.baseURL)

			if tt.testShorten {
				if tt.invalidOriginalURL != "" {
					repo.EXPECT().Save(gomock.Any()).Return(errors.New("некорректный url")).AnyTimes()
				} else {
					repo.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()
				}

				shortURL, err := svc.Shorten(tt.originalURL)
				if tt.wantShortenError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				if tt.wantEmptyShortUrl {
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

				if tt.wantResolveError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				if tt.wantResolveErrorIs != nil {
					assert.ErrorIs(t, err, tt.wantResolveErrorIs)
				}

				if tt.wantResolvedUrl != "" {
					assert.Equal(t, tt.wantResolvedUrl, result)
				}
			}
		})
	}

}
