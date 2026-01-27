package audit

// Event представляет событие аудита в системе сокращения URL.
type Event struct {
	// Timestamp — временная метка события в формате Unix timestamp.
	Timestamp int64 `json:"ts"`
	// Action — действие: shorten (создание) или follow (прохождение по ссылке).
	Action string `json:"action"`
	// UserID — идентификатор пользователя, если есть.
	UserID string `json:"user_id"`
	// URL — оригинальный (не сокращенный) URL.
	URL string `json:"url"`
}

// Константы типов действий аудита
const (
	// ActionShorten действие создания короткой ссылки
	ActionShorten = "shorten"
	// ActionFollow действие прохождения по короткой ссылке
	ActionFollow = "follow"
)
