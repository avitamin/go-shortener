package audit

// Observer определяет интерфейс для наблюдателя событий аудита (паттерн Observer).
// Реализации должны обрабатывать события аудита (запись в файл, отправка на сервер и т.д.).
type Observer interface {
	// Notify обрабатывает событие аудита.
	Notify(event Event) error
}
