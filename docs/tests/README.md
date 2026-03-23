# Тестирование

Обновлено: 2026-03-14

Документ описывает актуальный набор проверок для `go-shortener`.

## 1. Базовые проверки

### Unit + integration (Go)
```bash
go test ./...
```

### Линт
```bash
make lint
```

## 2. Транспортные проверки (HTTP + gRPC)

`cmd/testsuite` — отдельный CLI для end-to-end сценариев.

### Smoke
Проверяет жизнеспособность основных сценариев:
1. HTTP `POST /api/shorten`
2. gRPC `ExpandURL`
3. HTTP редирект `GET /{id}`

Запуск:
```bash
make smoke-test
# или
# go run ./cmd/testsuite -mode=smoke -http-base-url=http://localhost:8099 -grpc-address=localhost:9090
```

### Integration
Проверяет:
1. gRPC auth (без metadata -> `Unauthenticated`, с metadata -> успех)
2. Создание короткого URL через gRPC и резолв через HTTP

Запуск:
```bash
make integration-test
# или
# go run ./cmd/testsuite -mode=integration -http-base-url=http://localhost:8099 -grpc-address=localhost:9090
```

## 3. Параметры testsuite

Флаги `cmd/testsuite`:
- `-mode` (`smoke`|`integration`)
- `-http-base-url` (по умолчанию `http://localhost:8080`)
- `-grpc-address` (по умолчанию `localhost:9090`)
- `-tls` (использовать TLS для gRPC и HTTPS для HTTP)
- `-insecure-tls` (по умолчанию `true`, удобно для self-signed сертификата)
- `-timeout` (по умолчанию `20s`)

Пример TLS-прогона:
```bash
go run ./cmd/testsuite \
  -mode=smoke \
  -http-base-url=https://localhost:8099 \
  -grpc-address=localhost:9090 \
  -tls=true \
  -insecure-tls=true
```

## 4. Бенчмарки

Общие:
```bash
make bench
```

Точечные:
```bash
make bench-repo
make bench-service
make bench-handler
```

Профилирование памяти:
```bash
make profile-base
make profile-result
make profile-compare
make profile-analyze
```

## 5. CI-валидация

В GitHub Actions используются:
- Practicum autotests (`.github/workflows/shortenertest.yml`)
- `go vet` c `statictest` (`.github/workflows/statictest.yml`)

Для совместимости с Practicum используйте ветки вида `iter<number>`.

## 6. Минимальный регрессионный чек перед PR

1. `go test ./...`
2. `make lint`
3. `make smoke-test` (на поднятом приложении)
4. `make integration-test` (на поднятом приложении)
