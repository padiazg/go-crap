# Concepts

go-crap computes the CRAP score by combining two independent analyses and merging them:

## Pipeline Overview

```shell
go-crap scan
│
├── scan.Scan()                  — orchestrator (internal/scan)
│       ├── build exclude regex from --exclude flags
│       ├── coverage.Scan()      — discover Go modules, run go test -cover
│       │       │
│       │       └── coverage data per (file, func)
│       │
│       ├── complexity.Analyze() — walk AST, compute cyclomatic complexity
│       │       │
│       │       └── complexity data per (file, func)
│       │
│       ├── merge.Merge()        — join by (filepath, funcname)
│       │       │
│       │       └── merged entries with CC + Coverage
│       │
│       ├── score.Score()        — apply CRAP formula + missing policy
│       │       │
│       │       └── CRAP entries
│       │
│       └── applyFilters()       — sort descending, apply --min and --top
│
└── report.Format()              — table / json / github
        |
        └── output
```

## The Modules

Each module is independently testable:

| Module | Purpose |
| - | - |
| `internal/scan` | Orchestrates the pipeline: coverage, complexity, merge, score, filters |
| `internal/complexity` | AST walking to compute cyclomatic complexity (adapted from [gocyclo](https://github.com/fzipp/gocyclo), BSD-3-Clause) |
| `internal/coverage` | Module discovery + `go test -cover` profiling (adapted from [test-finder](https://github.com/padiazg/test-finder), MIT) |
| `internal/merge` | Join coverage and complexity by `(filepath, funcname)` using a double index |
| `internal/score` | CRAP formula + missing coverage policy |
| `internal/report` | Output formatters (table, JSON, GitHub annotations) |

## Important: Path Matching

Coverage and complexity produce paths in different formats. The merge layer uses a double index to match them — **never canonicalize relative paths against CWD**. See `internal/merge/index.go` and the test `TestMerge_RelativePathsNotResolvedAgainstCWD`.

## Function Name Matching

Coverage and complexity also produce function names in different formats. Coverage includes the receiver in the name (e.g. `Level.String`, `(*JSONFormatter).Format`), while complexity stores the receiver separately. The merge layer normalizes both sides to bare method names before matching, supporting pointer receivers (`*Type`), value receivers (`Type`), and plain functions.
