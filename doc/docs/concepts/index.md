# Concepts

go-crap computes the CRAP score by combining two independent analyses and merging them:

## Pipeline Overview

```
go-crap scan
│
├── coverage.Scan()     - discover Go modules, run go test -cover
│       |
|       └── coverage data per (file, func)
|
├── complexity.Analyze() - walk AST, compute cyclomatic complexity
|       |
|       └── complexity data per (file, func)
|
├── merge.Merge()       - join by (filepath, funcname)
|       |
|       └── merged entries with CC + Coverage
|
├── score.Score()       - apply CRAP formula + missing policy
|       |
|       └── CRAP entries
|
└── report.Format()     - table / json / github
        |
        └── output
```

## The Modules

Each module is independently testable:

| Module | Purpose |
|--------|---------|
| `internal/complexity` | AST walking to compute cyclomatic complexity (adapted from [gocyclo](https://github.com/fzipp/gocyclo), BSD-3-Clause) |
| `internal/coverage` | Module discovery + `go test -cover` profiling (adapted from [test-finder](https://github.com/padiazg/test-finder), MIT) |
| `internal/merge` | Join coverage and complexity by `(filepath, funcname)` using a double index |
| `internal/score` | CRAP formula + missing coverage policy |
| `internal/report` | Output formatters (table, JSON, GitHub annotations) |

## Important: Path Matching

Coverage and complexity produce paths in different formats. The merge layer uses a double index to match them - **never canonicalize relative paths against CWD**. See `internal/merge/index.go` and the test `TestMerge_RelativePathsNotResolvedAgainstCWD`.
