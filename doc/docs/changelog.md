# Changelog

All notable changes to go-crap will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
- **Filtering** - `--top N`, `--min score`, `--exclude glob`
- **CI integration** - `--fail-above` exits with code 1, `--format github` produces workflow annotations
- GitHub Actions CI/CD workflow (tests + CRAP check)
- GitHub Pages documentation deployment
