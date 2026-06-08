# Changelog

All notable changes to go-crap will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.3.0 - 2026-06-05

### Added

- New `sarif` output format — SARIF 2.1.0 compliant JSON for static analysis tools
- New `pr-comment` output format — markdown table formatted for pull request comments
- New `--verbose` flag — enables debug-level logging via `pkg/logger` and `pkg/slogger`
- New `--output` / `-o` flag — write output to a file instead of stdout

## v0.2.0 - 2026-06-04

### Added

- New `internal/scan` package — unified scan pipeline orchestrating coverage, complexity, merge, and score in a single call
- New `pkg/utils` package with `BuildExcludeRegex` and `MatchExclude` helpers
- Method receiver tracking — CRAP entries now include receiver info for struct methods
- `EntryList` wrapper type with `ThresholdExceeded()` method for CI integration
- `Skipped` field on `CRAPEntry` for functions excluded by `--missing skip` policy
- JSON output now uses `EntryList` structure with optional receiver field

### Changed

- `--exclude` flag now uses regex matching instead of glob matching
- Refactored `cmd/scan.go` to use new `internal/scan` package
- Coverage and complexity packages now accept `*regexp.Regexp` instead of `[]string` for exclude patterns
- Merge now constructs full function names from receiver + method name
- Report formatters now accept `*score.EntryList` instead of `[]score.CRAPEntry`
- GitHub formatter now includes function name (and receiver for methods) in warning messages

### Fixed

- Properly match function names with method receivers (e.g. `(*MyType).Method`)
- Handle zero-coverage entries correctly in merge step
- `runCoverTool` now runs in correct module directory (`modDir` set as `cmd.Dir`)
- Correct coverage path matching in CI environments — `buildSuffix` now uses 3 path components instead of 2, bridging Go module paths (`github.com/.../file.go`) and absolute filesystem paths (`/home/runner/.../go-crap/file.go`)
- Fix `normalizeFuncName` failing to strip value-receiver prefixes — methods like `Level.String` or `Logger.Debug` were not matched against coverage data, causing false 0% coverage reports for all value-receiver methods
- Fix merge discarding 0% coverage as "missing" — functions with known 0% coverage (coverage data present but no statements executed) are now correctly distinguished from functions with no coverage data at all

## v0.1.0 - 2026-06-02

### Added

- Initial release of go-crap
- `scan` command — analyze Go modules and calculate CRAP scores
- AST-based cyclomatic complexity via `internal/complexity` (adapted from [gocyclo](https://github.com/fzipp/gocyclo), BSD-3-Clause)
- Coverage profiling via `internal/coverage` (adapted from [test-finder](https://github.com/padiazg/test-finder), MIT)
- Double-index merge — joins coverage and complexity by `(filepath, funcname)` without CWD path resolution issues
- CRAP formula — `CC² × (1 - coverage/100)³ + CC`
- Missing coverage policy — `pessimistic` (default), `optimistic`, `skip`
- Output formatters:
  - `table` — human-readable with status symbols (✓, ▲, ✗) and coverage bars
  - `json` — structured output with schema URL
  - `github` — GitHub Actions workflow annotation format
- Filtering — `--top N`, `--min score`, `--exclude glob`
- CI integration — `--fail-above` exits with code 1, `--format github` produces workflow annotations
- GitHub Actions CI/CD workflow (tests + CRAP check)
- GitHub Pages documentation deployment
