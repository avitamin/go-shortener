# Руководство по профилированию и оптимизации

## Быстрый старт

### 1. Запуск бенчмарков

```bash
# Все бенчмарки
make bench

# Отдельные компоненты
make bench-repo      # Repository layer
make bench-service   # Service layer
make bench-handler   # Handler layer

# С сохранением результатов
make bench-save
```

### 2. Профилирование памяти

```bash
# Создать базовый профиль
make profile-base

# После оптимизации - создать результирующий профиль
make profile-result

# Сравнить профили
make profile-compare

# Анализировать текущий профиль
make profile-analyze
```

### 3. CPU профилирование

```bash
make bench-cpu

# Анализ
go tool pprof cpu.prof
```

## Детальный процесс оптимизации

### Шаг 1: Создание базового профиля

```bash
mkdir -p profiles
go test -bench=BenchmarkShorten$ -benchmem -memprofile=profiles/base.pprof ./internal/service -run=^$ -benchtime=5s
```

Результат:
```
BenchmarkShorten-16    	 2447185	      2423 ns/op	     531 B/op	       7 allocs/op
```

### Шаг 2: Анализ профиля

#### 2.1 Топ функций

```bash
go tool pprof -top profiles/base.pprof
```

Вывод:
```
      flat  flat%   sum%        cum   cum%
    1.54GB 84.48% 84.48%     1.54GB 84.48%  saveNoLock
    0.11GB  5.88% 90.36%     0.11GB  5.88%  GetAbsoluteShortURL
    0.09GB  5.13% 95.49%     0.09GB  5.13%  fmt.Sprintf
    0.03GB  1.64% 97.13%     0.06GB  3.19%  generateID
```

#### 2.2 Детали функций

```bash
go tool pprof -list=saveNoLock profiles/base.pprof
go tool pprof -list=GetAbsoluteShortURL profiles/base.pprof
go tool pprof -list=generateID profiles/base.pprof
```

### Шаг 3: Оптимизация кода

#### Проблема 1: Избыточное логирование

**До:**
```go
func (r *inMemoryStorage) saveNoLock(url model.URL) error {
    r.origByShort[url.Short] = url.Original
    r.shortByOrig[url.Original] = url.Short
    r.userIDByShort[url.Short] = url.UserID

    log.Println("сохранена запись ", url.Original, "->", url.Short)  // 98.50MB
    return nil
}
```

**После:**
```go
func (r *inMemoryStorage) saveNoLock(url model.URL) error {
    r.origByShort[url.Short] = url.Original
    r.shortByOrig[url.Original] = url.Short
    r.userIDByShort[url.Short] = url.UserID
    return nil
}
```

#### Проблема 2: Неэффективная конкатенация строк

**До:**
```go
func (s *ShortenerService) GetAbsoluteShortURL(short string) string {
    return s.Config.BaseURL + "/" + short  // 109.50MB
}
```

**После:**
```go
func (s *ShortenerService) GetAbsoluteShortURL(short string) string {
    var builder strings.Builder
    builder.Grow(len(s.Config.BaseURL) + 1 + len(short))
    builder.WriteString(s.Config.BaseURL)
    builder.WriteByte('/')
    builder.WriteString(short)
    return builder.String()
}
```

#### Проблема 3: Множественные аллокации буферов

**До:**
```go
func generateID() string {
    b := make([]byte, 6)  // Новая аллокация каждый раз
    io.ReadFull(rand.Reader, b)
    return base64.URLEncoding.EncodeToString(b)
}
```

**После:**
```go
var bytePool = sync.Pool{
    New: func() interface{} {
        b := make([]byte, 6)
        return &b
    },
}

func generateID() string {
    bp := bytePool.Get().(*[]byte)
    b := *bp
    defer bytePool.Put(bp)

    io.ReadFull(rand.Reader, b)
    return base64.URLEncoding.EncodeToString(b)
}
```

### Шаг 4: Повторное профилирование

```bash
go test -bench=BenchmarkShorten$ -benchmem -memprofile=profiles/result.pprof ./internal/service -run=^$ -benchtime=5s
```

Результат:
```
BenchmarkShorten-16    	 2507098	      2333 ns/op	     481 B/op	       4 allocs/op
```

### Шаг 5: Сравнение результатов

```bash
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

Вывод:
```
      flat  flat%   sum%        cum   cum%
 -109.50MB  5.88%  5.88%    -2.50MB  0.13%  GetAbsoluteShortURL
  -80.29MB  4.31%  4.45%   -80.29MB  4.31%  saveNoLock
  -30.50MB  1.64%  6.08%   -18.50MB  0.99%  generateID
```

Отрицательные значения показывают уменьшение использования памяти! ✅

## Анализ результатов

### Улучшения производительности

| Метрика | До | После | Изменение |
|---------|-----|-------|-----------|
| Время (ns/op) | 2423 | 2333 | **-3.7%** ⬇️ |
| Память (B/op) | 531 | 481 | **-9.4%** ⬇️ |
| Аллокации (allocs/op) | 7 | 4 | **-42.9%** ⬇️ |

### Снижение аллокаций памяти

- **GetAbsoluteShortURL**: -109.50MB
- **saveNoLock**: -80.29MB
- **generateID**: -30.50MB
- **Итого**: -105.79MB (5.68% от общего объема)

## Веб-интерфейс pprof

### Запуск веб-интерфейса

```bash
# Анализ одного профиля
go tool pprof -http=:8080 profiles/result.pprof

# Сравнение двух профилей
go tool pprof -http=:8080 -diff_base=profiles/base.pprof profiles/result.pprof
```

Откройте http://localhost:8080 в браузере для интерактивного анализа.

### Доступные представления

- **Top** - таблица функций по использованию памяти
- **Graph** - граф вызовов с размерами аллокаций
- **Flame Graph** - flame graph для визуализации
- **Peek** - детали отдельных функций
- **Source** - просмотр исходного кода с аннотациями

## Интерактивный режим

```bash
go tool pprof profiles/result.pprof
```

Полезные команды:
- `top` - топ функций
- `top -cum` - топ по кумулятивному использованию
- `list <function>` - исходный код функции с аннотациями
- `peek <function>` - вызовы и вызывающие функции
- `web` - граф в браузере (требует graphviz)
- `pdf` - экспорт в PDF (требует graphviz)
- `png` - экспорт в PNG (требует graphviz)
- `help` - справка

## Дополнительные инструменты

### benchstat - сравнение бенчмарков

```bash
# Установка
go install golang.org/x/perf/cmd/benchstat@latest

# Использование
go test -bench=. -benchmem ./internal/service > old.txt
# После оптимизации
go test -bench=. -benchmem ./internal/service > new.txt
# Сравнение
benchstat old.txt new.txt
```

### trace - анализ выполнения

```bash
# Создание trace
go test -bench=BenchmarkShorten$ -trace=trace.out ./internal/service

# Анализ
go tool trace trace.out
```

## Рекомендации по профилированию

### DO's ✅

- Профилируйте на репрезентативных данных
- Запускайте бенчмарки достаточно долго (`-benchtime=5s`)
- Используйте `-benchmem` для отслеживания аллокаций
- Профилируйте до и после оптимизации
- Используйте веб-интерфейс для визуализации
- Сохраняйте результаты для истории

### DON'Ts ❌

- Не оптимизируйте без профилирования
- Не профилируйте в Debug режиме
- Не делайте выводы на основе одного запуска
- Не профилируйте на виртуальных машинах (если возможно)
- Не игнорируйте количество аллокаций

## Автоматизация

### GitHub Actions

```yaml
- name: Run benchmarks
  run: |
    go test -bench=. -benchmem ./... | tee bench.txt

- name: Memory profiling
  run: |
    mkdir -p profiles
    go test -bench=BenchmarkShorten$ -memprofile=profiles/current.pprof ./internal/service

- name: Upload profiles
  uses: actions/upload-artifact@v2
  with:
    name: profiles
    path: profiles/
```

### Makefile интеграция

Все команды доступны через Makefile:

```bash
make help  # Список всех команд
```

## Дополнительные ресурсы

- [Go Blog: Profiling Go Programs](https://go.dev/blog/pprof)
- [Go Tool pprof Documentation](https://pkg.go.dev/runtime/pprof)
- [Benchmarking Best Practices](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [Memory Management in Go](https://go.dev/doc/effective_go#allocation_new)

## Заключение

Правильное профилирование - ключ к эффективной оптимизации. Всегда:
1. Измеряйте до оптимизации
2. Оптимизируйте
3. Измеряйте после оптимизации
4. Сравнивайте результаты

Помните: "Premature optimization is the root of all evil" - Donald Knuth
