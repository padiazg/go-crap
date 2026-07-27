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

### Mutation Validation

Coverage percentage alone doesn't guarantee test quality. A function with 100% coverage can still have blind spots — paths that are never executed by tests.

The `--mutation-report` flag validates coverage reliability by comparing mutation testing results (from [gremlins](https://github.com/go-gremlins/gremlins) or similar tools) against go-crap's coverage data. When a function has **lived** mutants, go-crap marks the coverage as unreliable and recalculates the CRAP score assuming 0% coverage.

This is surfaced in output via:
- ⚠ warning flag in `table` and `pr-comment` formatters
- Coverage-untrusted SARIF result in `sarif` format
- `mutation_score` and `coverage_untrusted` fields in `json` output
- "Unreliable Coverage" section in `pr-comment` output

### Detailed Mutation Details

The `--detailed` flag includes full mutation failure details in report output. Each survived mutant includes its type, line number, and the original/replacement code that was mutated.

- **JSON**: `mutation_details` array per entry with full mutant info
- **SARIF**: survived mutations with code diffs appended to warning messages
- **PR Comment**: `Survived Mutants` column with inline code snippets

### Baseline Comparison

The `--baseline` flag loads a previous JSON report and compares the current scan against it. This lets you track quality over time and catch regressions in CI.

When a baseline is provided:
- **pr-comment**: header shows Combined/Average CRAP with deltas vs baseline, plus "New Functions" and "Regressions" sections
- **table**: `Δ` column with per-function delta, delta footer lines
- **json**: per-entry `baseline_crap` and `delta` fields, `summary` object with baseline/compare stats (schema 1.1.0)
- **github**: summary `::notice::` annotation

Use `--fail-regression` to exit with code 1 when functions have regressed compared to the baseline. Adjust sensitivity with `--fail-regression-threshold`.

→ [Baseline Comparison Guide](integrations/baseline-comparison.md)

### Missing Coverage Policy

When a function has no coverage data, go-crap can handle it three ways:

- **pessimistic** (default) - assume 0% coverage, giving the maximum CRAP score
- **optimistic** - assume 100% coverage, giving the minimum CRAP score
- **skip** - exclude the function from results entirely

→ [Missing Coverage Policy](concepts/missing-policy.md)

### Output Formats

| Format | Flag | Use case |
| - | - | - |
| `table` | default | Human-readable terminal output with status symbols |
| `json` | `--format json` | Machine-readable output for CI pipelines |
| `github` | `--format github` | GitHub Actions workflow annotations |
| `sarif` | `--format sarif` | SARIF 2.1.0 for static analysis tools |
| `pr-comment` | `--format pr-comment` | Markdown table for pull request comments |

## Quick Start

```bash
# Install
curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh

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
curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh
```

→ [Installation Guide](getting-started/installation.md)
