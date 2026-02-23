// Package main contains project multichecker staticlint.
//
// # Run
//
// Local run:
//
//	go run ./cmd/staticlint ./...
//
// Build and run as go vet tool:
//
//	go build -o ./bin/staticlint ./cmd/staticlint
//	go vet -vettool=./bin/staticlint ./...
//
// # Included analyzers
//
// 1. Standard analyzers from golang.org/x/tools/go/analysis/passes:
// appends, asmdecl, assign, atomic, bools, buildtag, cgocall,
// composite, copylock, defers, directive, errorsas, framepointer,
// httpresponse, ifaceassert, loopclosure, lostcancel, nilfunc, printf,
// shift, sigchanyzer, slog, stdmethods, stdversion, stringintconv,
// structtag, testinggoroutine, tests, timeformat, unmarshal,
// unreachable, unsafeptr, unusedresult, waitgroup.
// They find suspicious constructions, API misuse, invalid tags,
// unreachable code, formatting mistakes and concurrency bugs.
//
// 2. All SA analyzers from staticcheck (honnef.co/go/tools/staticcheck).
// SA checks target correctness bugs: nil dereference risks,
// invalid standard library usage, incorrect slices/maps/channels logic,
// invalid regexp/time/http patterns, and similar defects.
//
// 3. Non-SA staticcheck class: S analyzers
// from honnef.co/go/tools/simple.
// They suggest simplifications that also reduce bug surface.
//
// 4. Extra public analyzers:
//   - bodyclose: checks that HTTP response bodies are closed.
//   - nilerr: reports returning nil after non-nil err checks.
//
// 5. Custom analyzer mainexit:
// forbids direct os.Exit calls in function main() of package main.
//
// # Purpose
//
// staticlint is the single static analysis entry point for this project.
// CI and local development should use this multichecker to keep code safe,
// consistent and aligned with project constraints.
package main
