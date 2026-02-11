# Профили памяти

Эта директория содержит профили памяти проекта, созданные с помощью pprof.

## Файлы

- `base.pprof` - Базовый профиль памяти (до оптимизации)
- `result.pprof` - Результирующий профиль памяти (после оптимизации)
- `comparison.txt` - Текстовое сравнение профилей
- `benchmark_results.txt` - Результаты бенчмарков

## Как использовать

### Создание нового базового профиля

```bash
make profile-base
```

или

```bash
go test -bench=BenchmarkShorten$ -benchmem -memprofile=profiles/base.pprof ./internal/service -run=^$ -benchtime=5s
```

### Создание результирующего профиля

После внесения оптимизаций:

```bash
make profile-result
```

или

```bash
go test -bench=BenchmarkShorten$ -benchmem -memprofile=profiles/result.pprof ./internal/service -run=^$ -benchtime=5s
```

### Сравнение профилей

```bash
make profile-compare
```

или

```bash
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

### Анализ профиля

```bash
make profile-analyze
```

или

```bash
go tool pprof -top profiles/result.pprof
go tool pprof -list=functionName profiles/result.pprof
```

### Интерактивный режим pprof

```bash
go tool pprof profiles/result.pprof
```

Доступные команды в интерактивном режиме:
- `top` - показать топ функций по использованию памяти
- `list <function>` - показать детали функции
- `web` - открыть граф в браузере (требует graphviz)
- `peek <function>` - показать вызовы функции
- `help` - справка по командам

## Веб-интерфейс

Для визуализации профиля в браузере:

```bash
go tool pprof -http=:8080 profiles/result.pprof
```

Затем откройте http://localhost:8080 в браузере.

## Сравнение через веб-интерфейс

```bash
go tool pprof -http=:8080 -diff_base=profiles/base.pprof profiles/result.pprof
```

## Примечание

Файлы `*.pprof` не добавляются в git согласно `.gitignore`.
Для воспроизведения результатов используйте команды `make profile-*`.
