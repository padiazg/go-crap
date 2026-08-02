# go-crap

[![Go Reference](https://pkg.go.dev/badge/github.com/padiazg/go-crap.svg)](https://pkg.go.dev/github.com/padiazg/go-crap)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CRAP analysis](https://github.com/padiazg/go-crap/actions/workflows/crap.yml/badge.svg?branch=master)](https://github.com/padiazg/go-crap/actions/workflows/crap.yml)

CRAP score calculator for Go projects. Calculates the CRAP score (cyclomatic complexity × coverage) for every function in a Go module. Inspired by [cargo-crap](https://github.com/Boehs/cargo-crap) for Rust.

## Installation

```shell
curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh
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

## Docker

Pull and run a pre-built image:

```shell
docker run --rm -v "$PWD:/code" ghcr.io/padiazg/go-crap scan
```

Registries: `docker.io/padiazg/go-crap`, `ghcr.io/padiazg/go-crap`

Available tags correspond to [releases](https://github.com/padiazg/go-crap/releases). Multi-arch images (linux/amd64, linux/arm64).

Build locally:

```shell
make docker-build
docker run --rm -v "$PWD:/code" go-crap:local scan
```

The image runs `go-crap scan /code` by default — it analyses whatever directory you mount at `/code`. Pass any flags directly:

```shell
docker run --rm -v "$PWD:/code" ghcr.io/padiazg/go-crap scan --top 10 --format table
```

## Usage

```shell
go-crap scan [pattern ...] [flags]
```

Scans Go packages matching the given patterns (defaults to `./...` — the whole module) and outputs a ranked table of functions by CRAP score.

Patterns use the same syntax as `go list` / `go build`: `./...` (recursive), `./internal/score` (single package), `github.com/x/y` (import path). Multiple patterns are accepted (e.g. `./internal/foo ./internal/bar`).

### Example

```shell
# Scan entire current module (default)
go-crap scan

# Scan specific package
go-crap scan ./internal/score

# Scan multiple packages
go-crap scan ./internal/score ./internal/coverage

# Show only the 10 worst offenders
go-crap scan --top 10

# Fail CI if any function exceeds threshold
go-crap scan --fail-above --threshold 30

# Exclude protobuf files (test files excluded by default)
go-crap scan --exclude 'pb/.*\.go'
```

### Flags

| Flag | Short | Description | Default |
| - | - | - | - |
| `--threshold` | `-t` | Score above which a function is marked as problematic | `30.0` |
| `--fail-above` | | Exit with code 1 if any function exceeds the threshold | `false` |
| `--fail-regression-ignore-covered` | | Exclude fully covered functions from regression failures (still shows them with `~`) | `false` |
| `--format` | `-f` | Output format: `table`, `json`, `github`, `sarif`, or `pr-comment` | `table` |
| `--top` | | Show only the N worst offenders (0 = all) | `0` |
| `--min` | | Hide entries below this score | `0` |
| `--missing` | | Policy for functions without coverage: `pessimistic`, `optimistic`, or `skip` | `pessimistic` |
| `--exclude` | | Exclude files matching this regex (repeatable). Use `.*` for any path depth. `_test.go` files are excluded by default | none |
| `--include-tests` | | Include `_test.go` files in analysis (overrides default exclude) | `false` |
| `--verbose` | | Enable verbose (debug-level) logging | `false` |
| `--progress` | | Show progress indicators (default: auto-detect terminal) | `false` |
| `--no-progress` | | Disable progress indicators | `false` |
| `--output` | `-o` | Output file path (default: stdout) | stdout |
| `--mutation-report` | | Path to gremlins JSON mutation report to validate coverage reliability | `""` |
| `--detailed` | | Include mutation failure details (original code, replacement, line) in report output | `false` |
| `--timeout` | | Timeout for the full scan (e.g. `30s`, `5m`, `1h30m`) | `10m0s` |
| `--coverage-profile` | | Supply an existing coverage profile instead of running `go test` | `""` |
| `--baseline` | | Path to a previous JSON report for baseline comparison | `""` |
| `--fail-regression` | | Exit code 1 when functions regressed vs baseline | `false` |
| `--fail-regression-threshold` | | Minimum delta to consider regression | `0.01` |
| `--show-unchanged` | | In baseline mode, also show functions whose CRAP score did not change (requires `--baseline`) | `false` |
| `--help` | `-h` | Help for scan | — |

### Progress Indicators

go-crap can show progress bars on stderr during long scans:

- **Default** — auto-detects terminal: enabled when stderr is a TTY, disabled when piped
- **`--progress`** — force-enable progress indicators
- **`--no-progress`** — force-disable progress indicators (useful in CI or when redirecting stderr)

Phases tracked: "Discovering modules", "Running coverage tests", "Analyzing complexity", "Processing results".

```shell
# Force progress in a non-interactive CI step
go-crap scan --progress

# Suppress bars when stderr is a TTY but you want clean output
go-crap scan --no-progress
```

### Output Formats

| Format | Description |
| - | - |
| `table` | Human-readable terminal output with status symbols and coverage bars |
| `json` | Structured output with `$schema` URL, suitable for CI pipelines |
| `github` | GitHub Actions workflow annotations (`::warning`) |
| `sarif` | SARIF 2.1.0 compliant JSON for static analysis tools |
| `pr-comment` | Markdown table formatted for pull request comments |

### Coverage Unavailable Warning

When a Go module fails to build or run tests, go-crap still parses the coverage
profile from any passing tests. The error is reported in all output formats;
functions exercised by passing tests appear with their real coverage, while
functions with no coverage data get a warning.

- `table` — coverage column shows `N/A ‼` with a footer listing unavailable modules
- `json` — `coverage` is `null` and `coverage_warning` contains the error message
- `github` — `::warning` annotation with the module error
- `sarif` — result with `RuleID: "go-crap/coverage-unavailable"`
- `pr-comment` — separate "Coverage Unavailable" section

### Example: SARIF output

```shell
go-crap scan --format sarif > crap.sarif
```

### Example: PR comment output

```shell
go-crap scan --format pr-comment > pr-comment.md
```

### Example: Mutation report validation

```shell
go-crap scan --mutation-report gremlins-report.json
```

When a function has **lived** mutants (mutations that survived because tests didn't catch them), go-crap marks the coverage as unreliable (`⚠`) and recalculates the CRAP score assuming 0% coverage. This catches functions that appear well-tested but have blind spots.

### Example: Detailed mutation output

```shell
go-crap scan --mutation-report gremlins-report.json --format json --detailed
```

The `--detailed` flag includes full mutation failure details in the output: `type`, `line`, `original_code`, and `replacement_code` for each survived mutant. In `json` format, these appear as a `mutation_details` array per entry. In `sarif` and `pr-comment` formats, survived mutations with code snippets are appended to the warning messages.

### Example: Baseline comparison

```shell
# Run a scan and save it as a baseline
go-crap scan --format json > baseline.json

# Compare against the baseline
go-crap scan --baseline baseline.json --format table

# Show all functions including unchanged ones
go-crap scan --baseline baseline.json --show-unchanged

# Enforce regression thresholds in CI
go-crap scan --baseline baseline.json --fail-regression --fail-regression-threshold 0.01
```

When a baseline is provided, every formatter shows deltas: the PR comment header displays combined/average CRAP with deltas, the table adds a `Δ` column, and JSON includes per-entry `baseline_crap` and `delta` fields.

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
  │   ├── merge.Merge()     — join by (filepath, funcname), propagate coverage warnings
  │   ├── score.Score()     — apply CRAP formula + missing policy
  │   ├── mutation.Annotate() — validate coverage with mutation testing (optional)
  │   └── report.Format()   — table / json / github / sarif / pr-comment
  │
  └── pkg/utils/            — regex helpers for --exclude patterns
```

- **`internal/scan`** — unified pipeline orchestrating the full scan flow (coverage → complexity → merge → score → filter → output)
- **`internal/complexity`** — AST walking to compute cyclomatic complexity (adapted from [gocyclo](https://github.com/fzipp/gocyclo), BSD-3-Clause)
- **`internal/coverage`** — module discovery + `go test -cover` profiling (adapted from [test-finder](https://github.com/padiazg/test-finder), MIT)
- **`internal/merge`** — double-index join of coverage and complexity data, propagates coverage-unavailable warnings from errored modules
- **`internal/score`** — CRAP formula + missing coverage policy + `EntryList` wrapper
- **`internal/mutation`** — parses gremlins JSON mutation reports and annotates CRAP entries with coverage reliability
- **`internal/report`** — output formatters (table, JSON, GitHub, SARIF, PR comment)
- **`pkg/logger`** — Logger interface and configuration types
- **`pkg/slogger`** — slog-backed Logger implementation
- **`pkg/utils`** — regex helper functions for exclude patterns

## CI Integration

```yaml
# .github/workflows/crap.yml
- run: go-crap scan --fail-above --threshold 30 --format github
```

- `--fail-above` exits with code 1 when any function exceeds the threshold
- `--fail-regression` exits with code 1 when functions regressed vs baseline
- `--fail-regression-ignore-covered` excludes fully covered functions from regression failures (still shows them with `~`)
- `--format github` emits `::warning` annotations that render as PR comments
- `--format sarif` outputs SARIF 2.1.0 for integration with code scanning tools
- `--format pr-comment` generates a markdown table for pull request comments
- `--output -o` writes results to a file instead of stdout
- `--mutation-report` validates coverage reliability against mutation testing results
- `--detailed` includes mutation failure details (code, line, type) in report output
- `--baseline` loads a previous JSON report for delta analysis
- `--show-unchanged` includes functions with no CRAP change in baseline output
- Coverage-unavailable warnings are emitted for modules where `go test` fails

### Badge

Add a status badge to your `README.md` to show the latest master analysis:

```markdown
[![CRAP analysis](https://github.com/YOUR_ORG/YOUR_REPO/actions/workflows/crap.yml/badge.svg?branch=master)](https://github.com/YOUR_ORG/YOUR_REPO/actions/workflows/crap.yml)
```

Requires the workflow to trigger on `push: branches: [master]` (not `branches-ignore`).

## Prior art and references

- [Savoia, A. & Evans, B. (2007). *The CRAP Metric.*](https://www.artima.com/weblogs/viewpost.jsp?thread=210575)
- [Crap4j](http://www.crap4j.org/) — the original Java implementation.
- [cargo-crap](https://github.com/minikin/cargo-crap) — Inspiration for this project

## License

This project is licensed under the [MIT License](LICENSE).

## Full Documentation

For a complete guide covering all flags, examples, and the CRAP formula in detail, see [the documentation site](https://padiazg.github.io/go-crap).
