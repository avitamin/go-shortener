# go-shortener

Обновлено: 2026-03-14

Сервис сокращения URL на Go с HTTP и gRPC интерфейсами.

## Быстрый старт

### Требования
- Go 1.24+
- Docker + Docker Compose (для локального PostgreSQL)

### Запуск через Docker Compose
```bash
make up
```

Приложение поднимется на `http://localhost:8099`, PostgreSQL на `localhost:54323`.

### Запуск локально через Go
```bash
make go-run
```

По умолчанию `make go-run` запускает:
- HTTP: `localhost:8099`
- gRPC: `localhost:9090`
- BASE_URL: `http://localhost:8099/`
- PostgreSQL: `postgres://postgres:postgres@localhost:54323/shortener?sslmode=disable`

## Основные команды

- `make build` — собрать `bin/shortener` с build-метаданными (`version/date/commit`)
- `make lint` — запустить `go run ./cmd/staticlint ./...`
- `go test ./...` — запустить unit/integration тесты Go
- `make smoke-test` — smoke проверка HTTP+gRPC через `cmd/testsuite`
- `make integration-test` — интеграционный сценарий cross-protocol
- `make up` / `make down` — поднять/остановить Docker Compose

Подробно по тестам: [docs/tests/README.md](docs/tests/README.md)

## Конфигурация

Конфигурация читается в порядке приоритета:
`env > flags > config file > defaults`

JSON-конфиг можно передать:
- флагом `-c` или `-config`
- переменной окружения `CONFIG`

### Параметры

| Флаг | Env | JSON | Описание | По умолчанию |
|---|---|---|---|---|
| `-a` | `SERVER_ADDRESS` | `server_address` | HTTP адрес | `localhost:8080` |
| `-ga` | `GRPC_SERVER_ADDRESS` | `grpc_server_address` | gRPC адрес | `localhost:9090` |
| `-b` | `BASE_URL` | `base_url` | Базовый URL для коротких ссылок | `http://localhost:8080` |
| `-f` | `FILE_STORAGE_PATH` | `file_storage_path` | Файловое хранилище | `./runtime/storage` |
| `-d` | `DATABASE_DSN` | `database_dsn` | PostgreSQL DSN | пусто |
| `-s` | `ENABLE_HTTPS` | `enable_https` | Включить TLS для HTTP и gRPC | `false` |
| `-t` | `TRUSTED_SUBNET` | `trusted_subnet` | CIDR для `/api/internal/stats` | пусто |
| `-audit-file` | `AUDIT_FILE` | `audit_file` | Аудит в файл | пусто |
| `-audit-url` | `AUDIT_URL` | `audit_url` | Аудит во внешний HTTP endpoint | пусто |
| — | `SECRET_KEY` | `secret_key` | HMAC-ключ подписи user token | `secret_key` |

Пример запуска с конфигом:
```bash
go run ./cmd/shortener -c=./runtime/config.json
```

## HTTP API

### Эндпоинты
- `POST /` — сократить URL (`text/plain`)
- `GET /{id}` — редирект на оригинальный URL
- `POST /api/shorten` — сократить URL (`application/json`)
- `POST /api/shorten/batch` — пакетное сокращение
- `GET /api/user/urls` — ссылки текущего пользователя
- `DELETE /api/user/urls` — асинхронное удаление ссылок пользователя
- `GET /ping` — health-check backend хранилища
- `GET /api/internal/stats` — статистика (`{"urls":N,"users":M}`), доступ только из `TRUSTED_SUBNET` по `X-Real-IP`

### Пример JSON сокращения
```bash
curl -i -X POST 'http://localhost:8099/api/shorten' \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}'
```

### Пример internal stats
```bash
curl -i 'http://localhost:8099/api/internal/stats' \
  -H 'X-Real-IP: 127.0.0.1'
```

Важно: без `TRUSTED_SUBNET` endpoint вернёт `403 Forbidden`.

## gRPC API

Proto: [api/proto/shortener.proto](api/proto/shortener.proto)

Сервис `ShortenerService`:
- `ShortenURL(URLShortenRequest) returns (URLShortenResponse)`
- `ExpandURL(URLExpandRequest) returns (URLExpandResponse)`
- `ListUserURLs(google.protobuf.Empty) returns (UserURLsResponse)`

### Авторизация в gRPC

Метод `ListUserURLs` требует metadata `authorization`:
- `Bearer <userID|signature>`
- либо `<userID|signature>`

`ShortenURL` и `ExpandURL` работают и без metadata.

Пример:
```bash
grpcurl -plaintext \
  -H 'authorization: Bearer <userID|signature>' \
  localhost:9090 shortener.ShortenerService/ListUserURLs
```

## TLS / HTTPS

При `-s` или `ENABLE_HTTPS=true`:
- HTTP и gRPC запускаются с TLS;
- сертификат генерируется self-signed на каждый старт процесса;
- браузер/клиент может показывать предупреждение о недоверенном сертификате.

Практика для локальной проверки — включать insecure TLS у клиентов (`cmd/testsuite` делает это по умолчанию).

## Хранилище

Выбор backend по конфигу:
1. Если задан `DATABASE_DSN`, используется PostgreSQL (+ миграции из `migrations/`).
2. Иначе, если задан `FILE_STORAGE_PATH`, используется файловое хранилище.
3. Иначе используется in-memory репозиторий.

## Документация по разделам

- Архитектура: [QWEN.md](QWEN.md)
- Тестирование: [docs/tests/README.md](docs/tests/README.md)
- Docker: [docker/README.md](docker/README.md)
- Правила для агентов и разработки: [AGENTS.md](AGENTS.md)
