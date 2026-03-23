# Имя бинарника
BINARY_NAME=bin/shortener
BUILD_VERSION?=dev
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS=-X 'main.buildVersion=$(BUILD_VERSION)' -X 'main.buildDate=$(BUILD_DATE)' -X 'main.buildCommit=$(BUILD_COMMIT)'
LOCAL_HTTP_ADDR?=localhost:8099
LOCAL_GRPC_ADDR?=localhost:9090
LOCAL_BASE_URL?=http://$(LOCAL_HTTP_ADDR)/
LOCAL_DB_DSN?=postgres://postgres:postgres@localhost:54323/shortener?sslmode=disable
TESTSUITE_HTTP_BASE_URL?=http://$(LOCAL_HTTP_ADDR)
TESTSUITE_GRPC_ADDR?=$(LOCAL_GRPC_ADDR)

# Цель по умолчанию
.DEFAULT_GOAL := help

# ===== Команды =====

## 🔧 Сборка Go бинарника локально
build:
	@echo "🛠️  Building Go binary..."
	@mkdir -p bin
	go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/shortener

## 🔍 Запуск project multichecker
lint:
	@echo "🔍 Running staticlint multichecker..."
	@go run ./cmd/staticlint ./...

## 📦 Генерация protobuf/gRPC кода
proto:
	@echo "📦 Generating protobuf and gRPC code..."
	protoc \
		--go_out=. \
		--go_opt=module=github.com/avitamin/go-shortener \
		--go_opt=default_api_level=API_OPAQUE \
		--go-grpc_out=. \
		--go-grpc_opt=module=github.com/avitamin/go-shortener \
		api/proto/shortener.proto

## ✅ Smoke test (requires running app)
smoke-test:
	@echo "✅ Running smoke testsuite..."
	@go run ./cmd/testsuite -mode=smoke -http-base-url=$(TESTSUITE_HTTP_BASE_URL) -grpc-address=$(TESTSUITE_GRPC_ADDR)

## 🧪 Integration test (requires running app)
integration-test:
	@echo "🧪 Running integration testsuite..."
	@go run ./cmd/testsuite -mode=integration -http-base-url=$(TESTSUITE_HTTP_BASE_URL) -grpc-address=$(TESTSUITE_GRPC_ADDR)

## 🧪 Обновление GoMock-ов
mocks:
	@echo "🧪 Generating mocks..."
	@mockgen -source=internal/repository/repository.go -destination=internal/repository/mock/repository.mock.go -package=mock

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
	@docker exec -it postgres_db psql -U postgres -d shortener

go-run:
	@go run -ldflags="$(LDFLAGS)" ./cmd/shortener -a=$(LOCAL_HTTP_ADDR) -ga=$(LOCAL_GRPC_ADDR) -b=$(LOCAL_BASE_URL) -d=$(LOCAL_DB_DSN)

## 📊 Запуск всех бенчмарков
bench:
	@echo "📊 Running all benchmarks..."
	@go test -bench=. -benchmem ./...

## 💾 Запуск бенчмарков с сохранением результатов
bench-save:
	@echo "💾 Running benchmarks and saving results..."
	@go test -bench=. -benchmem ./... | tee bench-$$(date +%Y%m%d-%H%M%S).txt

## 📈 Бенчмарки repository
bench-repo:
	@echo "📈 Running repository benchmarks..."
	@go test -bench=. -benchmem ./internal/repository

## 🎯 Бенчмарки service
bench-service:
	@echo "🎯 Running service benchmarks..."
	@go test -bench=. -benchmem ./internal/service

## 🌐 Бенчмарки handler
bench-handler:
	@echo "🌐 Running handler benchmarks..."
	@go test -bench=. -benchmem ./internal/handler

## 🔥 Бенчмарки с CPU профилированием
bench-cpu:
	@echo "🔥 Running benchmarks with CPU profiling..."
	@go test -bench=. -benchmem -cpuprofile=cpu.prof ./internal/repository
	@echo "Profile saved to cpu.prof. Open with: go tool pprof cpu.prof"

## 💾 Бенчмарки с памятью профилированием
bench-mem:
	@echo "💾 Running benchmarks with memory profiling..."
	@go test -bench=. -benchmem -memprofile=mem.prof ./internal/repository
	@echo "Profile saved to mem.prof. Open with: go tool pprof mem.prof"

## 🔬 Создание базового профиля памяти
profile-base:
	@echo "🔬 Creating base memory profile..."
	@mkdir -p profiles
	@go test -bench=BenchmarkShorten$$ -benchmem -memprofile=profiles/base.pprof ./internal/service -run=^$$ -benchtime=5s
	@echo "✓ Base profile saved to profiles/base.pprof"

## 🔬 Создание результирующего профиля памяти
profile-result:
	@echo "🔬 Creating result memory profile..."
	@mkdir -p profiles
	@go test -bench=BenchmarkShorten$$ -benchmem -memprofile=profiles/result.pprof ./internal/service -run=^$$ -benchtime=5s
	@echo "✓ Result profile saved to profiles/result.pprof"

## 📊 Сравнение профилей памяти
profile-compare:
	@echo "📊 Comparing memory profiles..."
	@go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof

## 🔍 Анализ текущего профиля памяти
profile-analyze:
	@echo "🔍 Analyzing memory profile..."
	@go tool pprof -top profiles/result.pprof

## 🧾 Справка по доступным командам
help:
	@echo ""
	@echo "Доступные команды:"
	@echo "  make build          - Сборка Go бинарника локально"
	@echo "  make mocks          - Генерация GoMock-ов"
	@echo "  make up             - Запуск Docker Compose (Go + PostgreSQL)"
	@echo "  make down           - Остановка контейнеров"
	@echo "  make clean          - Полная очистка контейнеров, образов и томов"
	@echo "  make logs           - Просмотр логов приложения"
	@echo "  make psql           - Подключение к базе PostgreSQL внутри контейнера"
	@echo ""
	@echo "Бенчмарки:"
	@echo "  make bench             - Запуск всех бенчмарков"
	@echo "  make bench-save        - Запуск бенчмарков с сохранением результатов"
	@echo "  make bench-repo        - Запуск бенчмарков repository"
	@echo "  make bench-service     - Запуск бенчмарков service"
	@echo "  make bench-handler     - Запуск бенчмарков handler"
	@echo "  make bench-cpu         - Запуск бенчмарков с CPU профилированием"
	@echo "  make bench-mem         - Запуск бенчмарков с памятью профилированием"
	@echo ""
	@echo "Профилирование памяти:"
	@echo "  make profile-base      - Создание базового профиля памяти"
	@echo "  make profile-result    - Создание результирующего профиля памяти"
	@echo "  make profile-compare   - Сравнение базового и результирующего профилей"
	@echo "  make profile-analyze   - Анализ текущего профиля памяти"
	@echo ""
