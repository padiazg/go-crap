# Changelog

All notable changes to go-crap will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.2.0 - [unreleased]

### Added

- New `internal/scan` package - unified scan pipeline orchestrating coverage, complexity, merge, and score in a single call
- New `pkg/utils` package with `BuildExcludeRegex` and `MatchExclude` helpers
- Method receiver tracking - CRAP entries now include receiver info for struct methods
- `EntryList` wrapper type with `ThresholdExceeded()` method for CI integration
- `Skipped` field on `CRAPEntry` for functions excluded by `--missing skip` policy
- JSON output now uses `EntryList` structure with optional receiver field

### Changed

- `--exclude` flag now uses regex matching instead of glob matching
- Refactored `cmd/scan.go` to use new `internal/scan` package
- Coverage and complexity packages now accept `*regexp.Regexp` instead of `[]string` for exclude patterns
- Merge now constructs full function names from receiver + method name
- Report formatters now accept `*score.EntryList` instead of `[]score.CRAPEntry`

### Fixed

- Properly match function names with method receivers (e.g. `(*MyType).Method`)
- Handle zero-coverage entries correctly in merge step
- `runCoverTool` now runs in correct module directory (`modDir` set as `cmd.Dir`)

## v0.1.0 - unreleased

### Added

- Initial release of go-crap
- `scan` command - analyze Go modules and calculate CRAP scores
- **AST-based cyclomatic complexity** via `internal/complexity` (adapted from [gocyclo](https://github.com/fzipp/gocyclo), BSD-3-Clause)
- **Coverage profiling** via `internal/coverage` (adapted from [test-finder](https://github.com/padiazg/test-finder), MIT)
- **Double-index merge** - joins coverage and complexity by `(filepath, funcname)` without CWD path resolution issues
- **CRAP formula** - `CC^2 x (1 - coverage/100)^3 + CC`
- **Missing coverage policy** - `pessimistic` (default), `optimistic`, `skip`
- **Output formatters**:
  - `table` - human-readable with status symbols (checkmark, A, x) and coverage bars
  - `json` - structured output with schema URL
  - `github` - GitHub Actions workflow annotation format
- **Filtering** - `--top N`, `--min score`, `--exclude regex`
- **CI integration** - `--fail-above` exits with code 1, `--format github` produces workflow annotations
- GitHub Actions CI/CD workflow (tests + CRAP check)
- GitHub Pages documentation deployment
