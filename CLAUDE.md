# go-crap

CLI tool that calculates the CRAP score (cyclomatic complexity × coverage)
for Go projects. Inspired by cargo-crap (Rust).

## Skills

Read the Go skills before writing any code:

go-testgen
golang-benchmark
golang-cli
golang-code-style
golang-concurrency
golang-context
golang-continuous-integration
golang-data-structures
golang-database
golang-dependency-injection
golang-dependency-management
golang-design-patterns
golang-documentation
golang-error-handling
golang-linter
golang-modernize
golang-naming
golang-observability
golang-performance
golang-popular-libraries
golang-project-layout
golang-safety
golang-security
golang-stay-updated
golang-stretchr-testify
golang-structs-interfaces
golang-troubleshooting

Apply skills as style guide, patterns, and conventions.
They take precedence over default preferences.

## Test Generation

Use go-testgen for all test scaffolding. Do not write *_test.go by hand.

    # Check what needs tests in a package
    go-testgen report ./internal/<pkg>/...

    # Generate scaffolding
    go-testgen gen ./internal/<pkg> <FuncName>
    go-testgen gen ./internal/<pkg> Receiver.Method
    go-testgen gen ./internal/<pkg> Receiver.Method --mock-from pkg.Interface

The generated style (checkXxxFn closures, before field, tests []struct table)
is the canonical style for this project. Do not deviate from it.

If go-testgen is not installed:
    go install github.com/padiazg/go-testgen/cmd/go-testgen@latest

Use the `gen-test-cases` and `closure-check-tests` skills as appropriate.

## Useful Commands

    go build ./...                          # compile
    go test ./...                           # all tests
    go test -short ./...                    # skip integration tests
    go test -v ./internal/score/...         # a specific package
    go-testgen report ./internal/<pkg>/...  # check pending test coverage

## Architecture

Modules are independent and independently testable:
- internal/complexity  — AST walking (adapted from gocyclo, BSD-3)
- internal/coverage    — module discovery + coverage (adapted from test-finder)
- internal/merge       — join by (filepath, funcname) with double index
- internal/score       — CRAP formula + missing policy
- internal/report      — formatters (table, json, github)
- cmd/                 — CLI Cobra

## Critical Issue: path matching

coverage and complexity produce paths in different formats.
See internal/merge/index.go — NEVER canonicalize relative paths against CWD.
The test TestMerge_RelativePathsNotResolvedAgainstCWD enforces this invariant.

## Attribution

internal/complexity/* adapted from github.com/fzipp/gocyclo (BSD-3-Clause).
internal/coverage/* adapted from github.com/padiazg/test-finder (MIT).
