FROM golang:alpine

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем go.mod и go.sum, скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем остальной код
COPY . .

# Собираем бинарник
RUN go build -o main ./cmd/shortener/main.go

# Запуск бинарника
CMD ["./main"]
