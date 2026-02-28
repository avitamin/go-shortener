# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go URL shortener service.
- `cmd/shortener`: main application entrypoint.
- `cmd/staticlint`: custom multichecker used for linting.
- `cmd/reset`: helper generator command.
- `internal/`: core application code (`handler`, `service`, `repository`, `middleware`, `config`, `audit`, `model`, `logger`, `pool`).
- `migrations/`: SQL migrations (`*.up.sql` / `*.down.sql`).
- `api/`: API-related docs.
- `profiles/`: pprof artifacts and profiling notes.
- `bin/`, `runtime/`: local runtime/build outputs.

Keep business logic in `internal/*`; keep `cmd/*` focused on wiring and startup.

## Build, Test, and Development Commands
Use the existing Make targets when possible:
- `make build`: build `bin/shortener` with version/date/commit ldflags.
- `make go-run`: run the app locally with a PostgreSQL DSN.
- `make up` / `make down`: start/stop Docker Compose stack.
- `make lint`: run the project multichecker (`go run ./cmd/staticlint ./...`).
- `go test ./...`: run all unit/integration tests.
- `make bench`, `make bench-service`, `make bench-handler`: run benchmarks.

## Coding Style & Naming Conventions
- Follow standard Go formatting: run `gofmt` (and `goimports` if imports change).
- Use tabs (Go default), lowercase package names, and `CamelCase` for exported identifiers.
- Keep files and symbols domain-oriented (`repository`, `service`, `handler`).
- Prefer small constructors in `cmd/*` and reusable logic in `internal/*`.

## Testing Guidelines
- Use Go `testing` with file names `*_test.go`.
- Test functions: `TestXxx`; benchmarks: `BenchmarkXxx`; examples: `Example_xxx`.
- Run `go test ./...` before opening a PR.
- For performance-sensitive changes, run the relevant `make bench-*` target and compare results.
- CI also runs static checks (`go vet` with `statictest`) and Practicum autotests.

## Commit & Pull Request Guidelines
- Recent history favors short, imperative commit subjects (often in Russian). Keep this style consistent and specific.
- Recommended commit format: `<area>: <action>` (example: `handler: fix gzip response headers`).
- PRs should include:
  - what changed and why,
  - linked issue/task,
  - validation evidence (commands run, e.g. `go test ./...`, `make lint`),
  - API examples for behavior changes (e.g., `curl` requests/responses).
- For Practicum CI compatibility, use branch names `iter<number>` when required.
