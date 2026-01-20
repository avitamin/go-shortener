package middleware

import (
	"net/http"

	"github.com/avitamin/go-shortener/internal/audit"
	"github.com/avitamin/go-shortener/internal/service"
	"go.uber.org/zap"
)

// auditResponseWriter оборачивает http.ResponseWriter для отслеживания статус кода и данных аудита
type auditResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	auditAction string
	auditURL    string
}

func (rw *auditResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *auditResponseWriter) Write(b []byte) (int, error) {
	// Если WriteHeader не был вызван явно, по умолчанию 200
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

// SetAuditData сохраняет данные для аудита в ResponseWriter
func SetAuditData(w http.ResponseWriter, action, url string) {
	if wrapper, ok := w.(*auditResponseWriter); ok {
		wrapper.auditAction = action
		wrapper.auditURL = url
	}
}

// AuditMiddleware middleware для аудита запросов
func AuditMiddleware(svc *service.ShortenerService, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Оборачиваем ResponseWriter для отслеживания статус кода и данных аудита
			wrapper := &auditResponseWriter{
				ResponseWriter: w,
				statusCode:     0,
			}

			// Выполняем следующий обработчик
			next.ServeHTTP(wrapper, r)

			// После выполнения обработчика проверяем, есть ли данные для аудита
			action := wrapper.auditAction
			url := wrapper.auditURL

			// Логируем только если есть данные для аудита и запрос успешен
			if action != "" && url != "" && isSuccessStatus(wrapper.statusCode) {
				userID, _ := service.GetUserIDFromContext(r.Context())

				// Вызываем соответствующий метод аудита
				switch action {
				case audit.ActionShorten:
					svc.Audit.LogShorten(userID, url)
				case audit.ActionFollow:
					svc.Audit.LogFollow(userID, url)
				default:
					log.Warn("Unknown audit action", zap.String("action", action))
				}
			}
		})
	}
}

// isSuccessStatus проверяет, является ли статус код успешным (2xx или 3xx)
func isSuccessStatus(code int) bool {
	return code >= 200 && code < 400
}
