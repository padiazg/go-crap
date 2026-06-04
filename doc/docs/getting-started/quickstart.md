# Quick Start

This guide walks through your first CRAP score scan in under two minutes.

## 1. Install go-crap

```shell
go install github.com/padiazg/go-crap@latest
```

## 2. Run a Scan

Point `go-crap scan` at a directory or package pattern:

```shell
go-crap scan
```

Example output:

```shell
┌───┬────────┬────┬───────────────────┬──────────┬─────────────────────────────────────┐
│   │ CRAP   │ CC │ COVERAGE          │ FUNCTION │ LOCATION                            |
├───┼────────┼────┼───────────────────┼──────────┼─────────────────────────────────────┤
| ✗ | 156.25 | 12 | ░░░░░░░░░░        | Analyze  │ internal/complexity/complexity.go:5 │
| ▲ │  42.50 |  5 │ ████░░░░░░        │ Scan     │ internal/coverage/coverage.go:28    │
| ✓ │   4.00 │  2 │ ██████████ 100.0% │ Execute  │ cmd/root.go:170                     │
└───┴────────┴────┴───────────────────┴──────────┴─────────────────────────────────────┘

1/3 function(s) exceed threshold CRAP 30.
```

Columns:

- **Status** - `✗` below threshold, `▲` between half-threshold and threshold, `✓` above threshold
- **CRAP** - the computed CRAP score
- **CC** - cyclomatic complexity
- **Coverage** - test coverage as a percentage with a visual bar
- **Function** - function name
- **Location** - file path and line number

## 3. Filter Results

### Show only the worst offenders

```shell
go-crap scan --top 10
```

### Hide entries below a minimum score

```shell
go-crap scan --min 10
```

### Exclude test files and testdata

```shell
go-crap scan --exclude '.*_test\.go' --exclude 'testdata/.*\.go'
```

### Exclude generated files at any depth

```shell
go-crap scan --exclude '\.pb\.go$' --exclude 'mock_'
```

## 4. Fail CI on High Scores

```shell
go-crap scan --fail-above --threshold 30
```

Exits with code 1 if any function's CRAP score exceeds 30.

## 5. Machine-Readable Output

### JSON

```shell
go-crap scan --format json
```

Output:

```json
{
  "$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
  "version": "1.0.0",
  "entries": [
    {
      "file": "internal/complexity/complexity.go",
      "package": "github.com/padiazg/go-crap/internal/complexity",
      "function": "Analyze",
      "line": 45,
      "cyclomatic": 12,
      "coverage": 0.0,
      "crap": 156.25
    }
  ]
}
```

### GitHub Actions annotations

```shell
go-crap scan --format github --threshold 15
::warning file=internal/coverage/scanner.go,line=149::internal/coverage/scanner.go:149 runTests CRAP score 19.6 (CC=7, cov=36.4%) exceeds threshold 15
::warning file=internal/complexity/analyze.go,line=136::internal/complexity/analyze.go:136 exprString CRAP score 19.1 (CC=6, cov=28.6%) exceeds threshold 15
::warning file=internal/complexity/analyze.go,line=62::internal/complexity/analyze.go:62 *analyzeData.analyzeDir CRAP score 17.1 (CC=14, cov=75.0%) exceeds threshold 15
::warning file=internal/coverage/parser.go,line=121::internal/coverage/parser.go:121 parseFileProfile CRAP score 15.8 (CC=15, cov=84.6%) exceeds threshold 15
```

Emits `::warning` annotations that GitHub Actions renders as PR comments.

## 6. Control Missing Coverage Policy

When a function has no coverage data, decide how to handle it:

```shell
# Assume 0% coverage (default) - worst case
go-crap scan --missing pessimistic

# Assume 100% coverage - best case
go-crap scan --missing optimistic

# Skip functions with no coverage
go-crap scan --missing skip
```
