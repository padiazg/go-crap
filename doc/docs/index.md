# go-crap

**go-crap** is a CLI tool that calculates the CRAP score (cyclomatic complexity x coverage) for Go projects. It walks the AST to compute cyclomatic complexity, merges it with test coverage data from `go test -cover`, and produces a CRAP score per function.

Inspired by [cargo-crap](https://github.com/Boehs/cargo-crap) for Rust.

## What It Does

Point it at a directory - it scans all Go modules, computes complexity, reads coverage, and outputs a ranked table of functions by CRAP score:

```bash
go-crap scan
```

## Key Concepts

### CRAP Score

The CRAP score measures how expensive a function is to test. It combines two factors:

- **Cyclomatic complexity (CC)** - how many independent paths through the function
- **Test coverage** - how much of the function is exercised by tests

A high CRAP score means the function is complex *and* poorly tested - it's a prime candidate for refactoring or more tests.

→ [CRAP Score Formula](concepts/crap-score.md)

### Missing Coverage Policy

When a function has no coverage data, go-crap can handle it three ways:

- **pessimistic** (default) - assume 0% coverage, giving the maximum CRAP score
- **optimistic** - assume 100% coverage, giving the minimum CRAP score
- **skip** - exclude the function from results entirely

→ [Missing Coverage Policy](concepts/missing-policy.md)

### Output Formats

| Format | Flag | Use case |
|--------|------|----------|
| `table` | default | Human-readable terminal output with status symbols |
| `json` | `--format json` | Machine-readable output for CI pipelines |
| `github` | `--format github` | GitHub Actions workflow annotations |

## Quick Start

```bash
# Install
go install github.com/padiazg/go-crap@latest

# Scan a project
go-crap scan

# Show only the 10 worst offenders
go-crap scan --top 10

# Fail CI if any function exceeds threshold
go-crap scan --fail-above --threshold 30
```

→ [Full Quick Start](getting-started/quickstart.md)

## Installation

```bash
go install github.com/padiazg/go-crap@latest
```

→ [Installation Guide](getting-started/installation.md)
