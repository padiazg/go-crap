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

### `--fail-regression-ignore-covered`

Exclude fully covered functions from regression failures (default: `false`).

```bash
go-crap scan --baseline baseline.json --fail-regression --fail-regression-ignore-covered
```

When enabled, functions with coverage ≥ 99.95% that have regressed are excluded from triggering the exit code 1 failure. They are still visible in the PR comment output with a `~` symbol.

This is useful when you want to enforce regression detection on partially covered functions while acknowledging that fully tested code shouldn't count as a failure.

## CI Example: GitHub Actions

The recommended setup is a single workflow with three jobs: one that generates a baseline
on `master`, a test job, and a PR comparison job that fetches the latest baseline.

### Full workflow

```yaml
name: crap
on:
  push:
    branches: [main, master]
  pull_request:

permissions:
  contents: read

jobs:
  baseline:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/master'
    steps:
      - uses: actions/checkout@v8
      - uses: actions/setup-go@v7
        with:
          go-version: '1.23'
          cache: true
      - name: Install go-crap
        run: curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh
      - name: Generate baseline
        run: go-crap scan --format json --output crap-current.json
      - uses: actions/upload-artifact@v8
        with:
          name: crap-baseline
          path: crap-current.json

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v8
      - uses: actions/setup-go@v7
        with:
          go-version: '1.23'
          cache: true
      - run: go test -race -count=1 ./...
      - name: Lint
        uses: golangci/golangci-lint-action@v9
        with:
          version: latest

  crap:
    runs-on: ubuntu-latest
    needs: test
    if: always() && needs.test.result == 'success'
    permissions:
      contents: read
      actions: read
    steps:
      - uses: actions/checkout@v8
      - uses: actions/setup-go@v7
        with:
          go-version: '1.23'
          cache: true
      - name: Install go-crap
        run: curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh

      - name: Find latest master baseline run
        id: get-run
        run: |
          RUN_ID=$(gh run list \
            --workflow crap.yml \
            --branch master \
            --status success \
            --json databaseId \
            --jq '.[0].databaseId' \
            --limit 1)
          echo "run_id=$RUN_ID" >> "$GITHUB_OUTPUT"
        env:
          GH_TOKEN: ${{ github.token }}
        continue-on-error: true

      - name: Download baseline
        uses: actions/download-artifact@v8
        with:
          name: crap-baseline
          path: baseline
          run-id: ${{ steps.get-run.outputs.run_id }}
          github-token: ${{ secrets.GITHUB_TOKEN }}
        continue-on-error: true

      - name: Check for regressions
        if: github.event_name == 'pull_request' && hashFiles('baseline/crap-current.json') != ''
        run: go-crap scan --baseline baseline/crap-current.json --fail-regression --fail-regression-ignore-covered

      - name: Generate PR comment
        if: github.event_name == 'pull_request'
        run: |
          if [ -f baseline/crap-current.json ]; then
            go-crap scan --baseline baseline/crap-current.json --format pr-comment --threshold 30 --output pr-comment.md
          else
            go-crap scan --format pr-comment --threshold 30 --output pr-comment.md
          fi

      - name: Save PR number
        if: github.event_name == 'pull_request'
        run: echo "${{ github.event.pull_request.number }}" > pr-number.txt

      - name: Upload PR comment artifacts
        uses: actions/upload-artifact@v8
        with:
          name: crap-comment
          path: |
            pr-comment.md
            pr-number.txt
          if-no-files-found: ignore
```

The `baseline` job only runs on push to `master`. For PRs, the `crap` job uses `gh run list` to
find the latest successful baseline run from `master` and downloads it. If no baseline exists
yet (e.g. first PR), the `crap` job continues with `--baseline` omitted.

### Fork-safe PR comment

The CI job's `GITHUB_TOKEN` is read-only on fork PRs. To post comments on forks,
use a separate workflow triggered by `workflow_run`:

```yaml
name: post PR comment
on:
  workflow_run:
    workflows: [crap]
    types: [completed]

permissions:
  pull-requests: write
  actions: read

jobs:
  comment:
    name: Post or update PR comment
    runs-on: ubuntu-latest
    if: github.event.workflow_run.event == 'pull_request'
    steps:
      - name: Download PR comment artifact
        uses: actions/download-artifact@v8
        with:
          name: crap-comment
          path: .
          run-id: ${{ github.event.workflow_run.id }}
          github-token: ${{ secrets.GITHUB_TOKEN }}
        continue-on-error: true

      - name: Post or update PR comment
        uses: actions/github-script@v9
        with:
          script: |
            const fs = require('fs');
            if (!fs.existsSync('pr-comment.md') || !fs.existsSync('pr-number.txt')) return;
            const body = fs.readFileSync('pr-comment.md', 'utf8');
            const prNumber = parseInt(fs.readFileSync('pr-number.txt', 'utf8').trim(), 10);
            if (!prNumber) return;
            const marker = '<!-- go-crap-report -->';
            const { data: comments } = await github.rest.issues.listComments({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: prNumber,
            });
            const existing = comments.find(c => c.body.startsWith(marker));
            if (existing) {
              await github.rest.issues.updateComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                comment_id: existing.id,
                body,
              });
            } else {
              await github.rest.issues.createComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                issue_number: prNumber,
                body,
              });
            }
```

The `<!-- go-crap-report -->` marker (emitted by `--format pr-comment`) ensures the
comment is updated in place on subsequent pushes instead of creating duplicates.

### Combined threshold + regression check

```yaml
      - name: Check CRAP quality
        run: >-
          go-crap scan
          --fail-above --threshold 30
          --baseline baseline/crap-current.json
          --fail-regression
          --fail-regression-ignore-covered
          --fail-regression-threshold 5.0
```

This fails if any function exceeds threshold 30 **or** if any function regressed by more than 5.0 vs baseline (excluding fully covered functions).
