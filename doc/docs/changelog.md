# Changelog

All notable changes to go-crap will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.4.1 - 2026-06-18

### Added

- Coverage unavailable warning — propagates `go test` failure messages through merge → score → formatters
- `CoverageWarning` field on `MergedEntry` and `CRAPEntry` — set when coverage data is unavailable due to module-level test errors
- `table` formatter: coverage column shows `N/A ‼` with deduplicated footer warning when coverage unavailable
- `json` formatter: `coverage` is `null` and `coverage_warning` field populated when coverage unavailable
- `github` formatter: emits `::warning` annotation when coverage is unavailable
- `sarif` formatter: new result with `RuleID: "go-crap/coverage-unavailable"` when coverage unavailable
- `pr-comment` formatter: "Coverage Unavailable" section listing affected functions
- Issue templates (bug report + feature request)
- Dependabot configuration
- `.golangci.yml` with full linter configuration
- `internal/scan/entries.go` with `EntryList` test helpers
- Extensive test coverage across all packages
- Mutation testing robustness improvements across all internal packages
- `permissions: read-all` to GitHub Actions workflow

### Changed

- Refactored `cmd/scan.go` output pipeline to use config struct
- Refactored `internal/scan/scan.go` with `Options` struct and helper extraction
- DRY refactors across internal packages
- Dead code removal
- Doc comment cleanup across internal packages
- Bumped golangci-lint config version to v1.65.0
- Bumped golangci-lint-action from v6 to v9
- Removed dead code in `pkg/slogger` and `pkg/dummylogger`

### Fixed

- `crappy[:25]` slice bounds panic when filtered entries < 25 items
- Linting issues across codebase
- Typo `ThresholdExeeded` → `ThresholdExceeded`
- Typo in version command description ("TestGen" → "go-crap")
- Logic bugs in merge, score, mutation, report, and scan packages
- Typos and embarrassment fixes across codebase

## v0.4.0 - 2026-06-08

### Added

- New `--mutation-report` flag — path to a gremlins JSON mutation report to validate coverage reliability
- New `internal/mutation` package — parses gremlins mutation reports and annotates CRAP entries
- New `CoverageUntrusted` field on `CRAPEntry` — set to `true` when lived mutants are found in a function
- New `MutationScore` field on `CRAPEntry` — killed/(killed+lived) ratio for the function
- New `EffectiveCRAP` field on `CRAPEntry` — CRAP score recalculated with 0% coverage when `CoverageUntrusted` is `true`
- Mutation score included in JSON output (`mutation_score`, `coverage_untrusted`, `effective_crap` fields)
- Coverage warning flag (⚠) in `table` and `pr-comment` formatters when coverage is unreliable
- Coverage-untrusted warnings in `github` and `sarif` formatters (SARIF adds a second `coverage-untrusted` result)
- New "Unreliable Coverage" section in `pr-comment` output listing affected functions with mutation scores
- New `--detailed` flag — includes mutation failure details (original/replacement code, line, type) in report output
- New `MutationDetail` struct on `CRAPEntry` — stores survived mutant details when mutation report is provided
- JSON output includes `mutation_details` array per entry when `--detailed` is set, with `type`, `mutator_name`, `file`, `line`, `status`, `original_text`, and `replacement_text` fields
- SARIF appends survived mutation details (type, line, code diff) to warning messages when `--detailed`
- PR Comment adds `Survived Mutants` column with code snippets when `--detailed`
- New `OriginalCode` and `ReplacementCode` fields on mutation `Mutant` struct, parsed from Gremlins JSON report

### Changed

- `ThresholdExceeded()` now uses `EffectiveCRAP` instead of `CRAP` for threshold comparison
- Filtering (`--top`, `--min`) and sorting now use `EffectiveCRAP` when mutation report is provided
- `--fail-above` now checks `EffectiveCRAP` against the threshold

### Fixed

- Functions with lived mutants now show their true risk level via `EffectiveCRAP` (CRAP at 0% coverage)

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
