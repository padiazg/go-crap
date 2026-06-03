# Quick Start

This guide walks through your first CRAP score scan in under two minutes.

## 1. Install go-crap

```bash
go install github.com/padiazg/go-crap@latest
```

## 2. Run a Scan

Point `go-crap scan` at a directory or package pattern:

```bash
go-crap scan
```

Example output:

```
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

```bash
go-crap scan --top 10
```

### Hide entries below a minimum score

```bash
go-crap scan --min 10
```

### Exclude files matching a glob

```bash
go-crap scan --exclude '*/testdata/*' --exclude '*_test.go'
```

## 4. Fail CI on High Scores

```bash
go-crap scan --fail-above --threshold 30
```

Exits with code 1 if any function's CRAP score exceeds 30.

## 5. Machine-Readable Output

### JSON

```bash
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

```bash
go-crap scan --format github
```

Emits `::warning` annotations that GitHub Actions renders as PR comments.

## 6. Control Missing Coverage Policy

When a function has no coverage data, decide how to handle it:

```bash
# Assume 0% coverage (default) - worst case
go-crap scan --missing pessimistic

# Assume 100% coverage - best case
go-crap scan --missing optimistic

# Skip functions with no coverage
go-crap scan --missing skip
```
