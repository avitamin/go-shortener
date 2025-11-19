# Имя бинарника
BINARY_NAME=main

# Цель по умолчанию
.DEFAULT_GOAL := help

# ===== Команды =====

## 🔧 Сборка Go бинарника локально
build:
	@echo "🛠️  Building Go binary..."
	go build -o $(BINARY_NAME) .

## 🚀 Запуск docker-compose
up:
	@echo "🚀 Starting Docker containers..."
	docker-compose up -d --build

## 🧹 Остановка контейнеров
down:
	@echo "🧹 Stopping Docker containers..."
	docker-compose down

## 🧼 Удаление контейнеров, образов и томов
clean:
	@echo "🧽 Removing all containers, images, and volumes..."
	docker-compose down --rmi all --volumes --remove-orphans

## 📜 Просмотр логов приложения
logs:
	@docker-compose logs -f app

## 🧠 Проверка подключений к PostgreSQL
psql:
	@docker exec -it postgres_db psql -U postgres -d postgres

go-run:
	@go run ./cmd/shortener/main.go -a=localhost:8099 -b=http://localhost:8099/ -d=postgres://postgres:postgres@localhost:54323/shortener?sslmode=disable

## 🧾 Справка по доступным командам
help:
	@echo ""
	@echo "Доступные команды:"
	@echo "  make build     - Сборка Go бинарника локально"
	@echo "  make up        - Запуск Docker Compose (Go + PostgreSQL)"
	@echo "  make down      - Остановка контейнеров"
	@echo "  make clean     - Полная очистка контейнеров, образов и томов"
	@echo "  make logs      - Просмотр логов приложения"
	@echo "  make psql      - Подключение к базе PostgreSQL внутри контейнера"
	@echo ""
