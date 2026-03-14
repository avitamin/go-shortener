# Repository Guidelines

Updated: 2026-03-14

## Project Structure & Module Organization
This repository is a Go URL shortener service.
- `cmd/shortener`: main application entrypoint (HTTP + gRPC runtime).
- `cmd/testsuite`: smoke/integration CLI checks for HTTP+gRPC.
- `cmd/staticlint`: custom multichecker used for linting.
- `cmd/reset`: helper generator command.
- `internal/`: core application code (`handler`, `grpc`, `service`, `repository`, `middleware`, `config`, `audit`, `model`, `logger`, `pool`, `auth`, `transport`).
- `api/proto`: protobuf contract for gRPC.
- `migrations/`: SQL migrations (`*.up.sql` / `*.down.sql`).
- `profiles/`: pprof artifacts and profiling notes.
- `bin/`, `runtime/`: local runtime/build outputs.

Keep business logic in `internal/*`; keep `cmd/*` focused on wiring and startup.

## Build, Test, and Development Commands
Use existing `makefile` targets when possible:
- `make build`: build `bin/shortener` with version/date/commit ldflags.
- `make go-run`: run HTTP (`:8099`) + gRPC (`:9090`) locally with PostgreSQL DSN.
- `make up` / `make down`: start/stop Docker Compose stack.
- `make lint`: run the project multichecker (`go run ./cmd/staticlint ./...`).
- `go test ./...`: run all unit/integration tests.
- `make smoke-test`: run smoke scenario via `cmd/testsuite`.
- `make integration-test`: run integration scenario via `cmd/testsuite`.
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
- For transport regression checks, also run `make smoke-test` and `make integration-test` against a running app.
- For performance-sensitive changes, run relevant `make bench-*` targets and compare results.
- CI also runs static checks (`go vet` with `statictest`) and Practicum autotests.

## Commit & Pull Request Guidelines
- Recent history favors short, imperative commit subjects (often in Russian). Keep this style consistent and specific.
- Recommended commit format: `<area>: <action>` (example: `handler: fix gzip response headers`).
- PRs should include:
  - what changed and why,
  - linked issue/task,
  - validation evidence (commands run, e.g. `go test ./...`, `make lint`),
  - API examples for behavior changes (`curl`/`grpcurl` requests and responses).
- For Practicum CI compatibility, use branch names `iter<number>` when required.
