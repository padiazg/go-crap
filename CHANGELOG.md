# Changelog

All notable changes to go-crap will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.5.0 - 2026-07-28

### Added

- Partial coverage preserved on test failure — when `go test` fails in a module, go-crap still parses the coverage profile from any passing tests instead of discarding all coverage data
- `extractFailedTests` helper parses `--- FAIL: TestName` lines from stderr and logs failed test names at Warn level (use `--verbose` to see them)
- `--coverage-profile` flag — supply an existing coverage profile instead of running `go test` (collaboration with @mem)
- `--timeout` flag — configurable timeout for the scan command (collaboration with @matiasinsaurralde)
- `Profile` field on `ModuleCoverage` — stores the path to the coverage profile file
- Baseline comparison engine — load previous JSON reports for delta analysis
- New `--baseline <path>` flag — compare current CRAP against a previous report
- New `--fail-regression` flag — exit code 1 when functions regressed vs baseline
- New `--fail-regression-ignore-covered` flag — exclude fully covered functions from regression failures (still shows them with `~`)
- New `--fail-regression-threshold <float>` flag (default `0.01`) — minimum delta to trigger
- `Baseline` struct with `LoadBaseline()` — parses JSON reports into comparison-ready data
- `AnnotateWithBaseline()` — sets `BaselineCRAP` on each CRAP entry for delta tracking
- `ComputeSummaryWithBaseline()` — aggregate stats with baseline/compare deltas
- `FindRegressions()` — exported regression detection for CLI use
- `ErrRegression` sentinel error — distinct from threshold exceedance
- `BaselineCRAP` field on `CRAPEntry` — stores previous report CRAP value (-1 = new function)
- `Delta` field on `CRAPEntry` — CRAP delta from baseline

### Changed

- `runTests` no longer deletes the temp coverage profile on error; the profile is kept for `parseCoverProfile` and cleaned up after parsing
- `scanModule` no longer returns early on `runTests` failure; coverage is always parsed and partial data is returned alongside the error
- `Scan` preserves the full `ModuleCoverage` struct from `scanModule`, retaining `Functions` alongside the wrapped error
- JSON schema bumped to 1.1.0 — per-entry `baseline_crap`/`delta` fields, `summary` object with baseline/compare
- PR comment formatter: header shows Combined/Average CRAP with deltas vs baseline; ASCII badges (`[OK]`, `[!!]`, `[ERROR]`, `[NEW]`); "New Functions" and "Regressions" sections; summary table
- Table formatter: `Δ` column when baseline available; per-function delta with `+X ↑` / `-X ↓`; delta footer lines
- GitHub formatter: summary `::notice::` annotation when baseline provided
- CLI: `--fail-regression` without `--baseline` returns error
- JSON output: schema field updated to 1.1.0, new `summary` object with baseline/compare stats

### Fixed

- Coverage profile deduplication — when `go test -coverprofile` produces a merged profile from multiple test binaries, the same source blocks appeared multiple times. Entries are now deduplicated by `(startLine, endLine)` position using OR semantics (a block is covered if any test binary covered it), fixing inaccurate coverage percentages (reported by @corani)
- Coverage profile temp files now cleaned up after successful scan (collaboration with @matiasinsaurralde)
- Field alignment for better memory layout
- Tests added for `Spash` function

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
- JSON output now includes `mutation_details` array per entry when `--detailed` is set, with `type`, `mutator_name`, `file`, `line`, `status`, `original_text`, and `replacement_text` fields
- SARIF output appends survived mutation details (type, line, code diff) to warning messages when `--detailed`
- PR comment output adds `Survived Mutants` column with code snippets when `--detailed`
- New `OriginalCode` and `ReplacementCode` fields on mutation `Mutant` struct, parsed from Gremlins JSON report

### Changed

- `ThresholdExceeded()` now uses `EffectiveCRAP` instead of `CRAP` for threshold comparison
- `--fail-above` now checks `EffectiveCRAP` against the threshold

### Fixed

- Functions with lived mutants now show their true risk level via `EffectiveCRAP` (CRAP at 0% coverage)
- `--top` no longer drops functions flagged with lived mutation mutants — `CoverageUntrusted` entries always survive truncation alongside the top trusted entries
- `--min` no longer filters out `CoverageUntrusted` entries, even when their `CRAP` score is below the threshold
- `github` format now emits `::warning` for `CoverageUntrusted` entries regardless of threshold
- `pr-comment` format now always renders the "Unreliable Coverage" section when mutation report is provided, even when no function exceeds the CRAP threshold

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
