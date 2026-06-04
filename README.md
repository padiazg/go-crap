# go-crap

[![Go Reference](https://pkg.go.dev/badge/github.com/padiazg/go-crap.svg)](https://pkg.go.dev/github.com/padiazg/go-crap)
[![Go Report Card](https://goreportcard.com/badge/github.com/padiazg/go-crap)](https://goreportcard.com/report/github.com/padiazg/go-crap)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

CRAP score calculator for Go projects. Calculates the CRAP score (cyclomatic complexity × coverage) for every function in a Go module. Inspired by [cargo-crap](https://github.com/Boehs/cargo-crap) for Rust.

## Installation

```shell
go install github.com/padiazg/go-crap@latest
```

Or build from source:

```shell
git clone https://github.com/padiazg/go-crap.git
cd go-crap
go build -o go-crap .
```

Or use Brew:

```shell
brew tap padiazg/go-crap 
brew install go-crap
```

## Usage

```shell
go-crap scan [path] [flags]
```

Scans the Go module at the given path (defaults to `.`) and outputs a ranked table of functions by CRAP score.

### Example

```shell
# Scan current module
go-crap scan

# Scan a specific directory
go-crap scan ./internal/score

# Show only the 10 worst offenders
go-crap scan --top 10

# Fail CI if any function exceeds threshold
go-crap scan --fail-above --threshold 30

# Exclude test files and protobuf
go-crap scan --exclude '.*_test\.go' --exclude 'pb/.*\.go'
```

### Flags

| Flag | Short | Description | Default |
| - | - | - | - |
| `--threshold` | `-t` | Score above which a function is marked as problematic | `30.0` |
| `--fail-above` | | Exit with code 1 if any function exceeds the threshold | `false` |
| `--format` | `-f` | Output format: `table`, `json`, or `github` | `table` |
| `--top` | | Show only the N worst offenders (0 = all) | `0` |
| `--min` | | Hide entries below this score | `0` |
| `--missing` | | Policy for functions without coverage: `pessimistic`, `optimistic`, or `skip` | `pessimistic` |
| `--exclude` | | Exclude files matching this regex (repeatable). Use `.*` for any path depth. e.g. `.*_test\.go` to exclude all test files, `pb/.*\.go` to exclude protobuf files | none |

### Output Formats

| Format | Description |
| - | - |
| `table` | Human-readable terminal output with status symbols and coverage bars |
| `json` | Structured output with `$schema` URL, suitable for CI pipelines |
| `github` | GitHub Actions workflow annotations (`::warning`) |

## What is CRAP?

CRAP = **C**yclomatic **R**eadability **A**nd **P**redictability. It measures how expensive a function is to test.

$CRAP(CC, coverage) = CC^2 × \left(1 - \frac{coverage}{100}\right)^3 + CC$

A function with high cyclomatic complexity and low coverage scores the worst. A simple, fully tested function scores the best.

| CRAP Range | Meaning |
| - | - |
| 0 – 10 | Well-tested, simple function |
| 10 – 30 | Moderate complexity, should be tested |
| 30 – 50 | High CRAP — refactoring or more tests needed |
| 50+ | Critical — likely hard to test, complex logic |

## How It Works

```
go-crap scan
  │
  ├── scan.Scan()           — unified pipeline, discovers modules, filters, and ranks
  │   ├── coverage.Scan()   — discover Go modules, run go test -cover
  │   ├── complexity.Analyze() — walk AST, compute cyclomatic complexity
  │   ├── merge.Merge()     — join by (filepath, funcname) with receiver support
  │   ├── score.Score()     — apply CRAP formula + missing policy
  │   └── report.Format()   — table / json / github
  │
  └── pkg/utils/            — regex helpers for --exclude patterns
```

- **`internal/scan`** — unified pipeline orchestrating the full scan flow (coverage → complexity → merge → score → filter → output)
- **`internal/complexity`** — AST walking to compute cyclomatic complexity (adapted from [gocyclo](https://github.com/fzipp/gocyclo), BSD-3-Clause)
- **`internal/coverage`** — module discovery + `go test -cover` profiling (adapted from [test-finder](https://github.com/padiazg/test-finder), MIT)
- **`internal/merge`** — double-index join of coverage and complexity data, with method receiver support
- **`internal/score`** — CRAP formula + missing coverage policy + `EntryList` wrapper
- **`internal/report`** — output formatters (table, JSON, GitHub annotations)
- **`pkg/utils`** — regex helper functions for exclude patterns

## CI Integration

```yaml
# .github/workflows/crap.yml
- run: go-crap scan --fail-above --threshold 30 --format github
```

- `--fail-above` exits with code 1 when any function exceeds the threshold
- `--format github` emits `::warning` annotations that render as PR comments

## License

This project is licensed under the [MIT License](LICENSE).

## Full Documentation

For a complete guide covering all flags, examples, and the CRAP formula in detail, see [the documentation site](https://padiazg.github.io/go-crap).
