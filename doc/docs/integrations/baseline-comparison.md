# Baseline Comparison

Compare current CRAP scores against a previous report to detect regressions.

## Quick Start

Generate a baseline report, then compare future scans against it:

```bash
# Generate baseline
go-crap scan --format json --output baseline.json

# Compare against baseline
go-crap scan --baseline baseline.json --format pr-comment
```

## CLI Flags

### `--baseline <path>`

Path to a previous JSON report for comparison.

```bash
go-crap scan --baseline previous-report.json
```

When set, each formatter includes delta information:

- **Table**: `Δ` column showing per-function CRAP change
- **PR Comment**: header with deltas, Regressions section, New Functions section
- **JSON**: `baseline_crap`, `delta`, and `summary` object with baseline/compare fields
- **GitHub**: summary `::notice::` annotation

### `--fail-regression`

Exit with code 1 if any function's CRAP score regressed compared to baseline.

```bash
go-crap scan --baseline baseline.json --fail-regression
```

Requires `--baseline`. Combines with `--fail-above` -- either check triggers exit code 1.

### `--fail-regression-threshold <float>`

Minimum CRAP delta to consider a regression (default: `0.01`).

```bash
go-crap scan --baseline baseline.json --fail-regression --fail-regression-threshold 5.0
```

Only flags functions whose CRAP increased by more than the threshold.

## CI Example: GitHub Actions

### Generate baseline on main

```yaml
name: crap-baseline
on:
  push:
    branches: [main, master]

jobs:
  baseline:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true
      - name: Generate baseline
        run: go install github.com/padiazg/go-crap@latest
      - name: Run go-crap
        run: go-crap scan --format json --output crap-current.json
      - name: Upload baseline
        uses: actions/upload-artifact@v4
        with:
          name: crap-baseline
          path: crap-current.json
```

### Compare on PR

```yaml
name: crap-pr
on:
  pull_request:

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true
      - name: Download baseline
        uses: actions/download-artifact@v4
        with:
          name: crap-baseline
          path: baseline
      - name: Run go-crap
        run: go install github.com/padiazg/go-crap@latest
      - name: Check for regressions
        run: go-crap scan --baseline baseline/crap-current.json --fail-regression
      - name: Generate PR comment
        run: go-crap scan --baseline baseline/crap-current.json --format pr-comment --output pr-comment.md
      - name: Comment on PR
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const comment = fs.readFileSync('pr-comment.md', 'utf8');
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: comment
            });
```

### Combined threshold + regression check

```yaml
      - name: Check CRAP quality
        run: >-
          go-crap scan
          --fail-above --threshold 30
          --baseline baseline/crap-current.json
          --fail-regression
          --fail-regression-threshold 5.0
```

This fails if any function exceeds threshold 30 **or** if any function regressed by more than 5.0 vs baseline.
