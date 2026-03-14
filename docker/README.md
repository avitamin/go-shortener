# Docker и развёртывание

Обновлено: 2026-03-14

Документ описывает локальное развёртывание через `docker-compose.yaml`.

## Состав окружения

`docker-compose.yaml` поднимает 2 сервиса:
1. `app` — приложение `go-shortener`
2. `db` — PostgreSQL 16

## Быстрый запуск

```bash
make up
```

Проверить логи приложения:
```bash
make logs
```

Остановить окружение:
```bash
make down
```

Полная очистка (контейнеры/образы/тома):
```bash
make clean
```

## Порты и адреса

- HTTP сервиса: `http://localhost:8099`
- PostgreSQL: `localhost:54323`

Внутри compose-сети приложение стартует с:
- `SERVER_ADDRESS=:8080`
- `DATABASE_DSN=postgres://postgres:postgres@db:5432/shortener?sslmode=disable`
- `SECRET_KEY=secret_key`

## Особенности

- В compose-конфигурации наружу проброшен только HTTP порт.
- gRPC порт не проброшен. Для локальной проверки gRPC используйте запуск вне Docker (`make go-run`) или добавьте проброс порта в compose.

## Проверка доступности

После `make up`:
```bash
curl -i http://localhost:8099/ping
```

Ожидается `HTTP/1.1 200 OK`.

## Прод-развёртывание

Текущий compose ориентирован на локальную разработку:
- секреты заданы в явном виде;
- нет внешнего TLS termination;
- нет readiness/liveness probe и оркестрации.

Для production рекомендуется:
1. Выносить секреты в secret manager / environment провайдера.
2. Использовать внешний TLS termination (reverse proxy / ingress).
3. Настроить мониторинг, алерты и резервное копирование PostgreSQL.
4. Ограничить сетевые политики и доступ к БД.
