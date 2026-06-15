# CI Integrations

Integrate go-crap into your continuous integration pipeline to enforce CRAP score thresholds across all pull requests.

## Providers

- [GitHub Actions](github-actions.md) -- threshold enforcement, PR annotations, matrix builds, SARIF code scanning, fork-safe PR comments with mutation testing
- [Gitea](gitea.md) -- Gitea Actions with annotation support
- [GitLab CI](gitlab.md) -- quality stage with JSON artifact
- [Azure DevOps](azure-devops.md) -- pipeline with artifact upload
- [CircleCI](circleci.md) -- quality workflow
- [Jenkins](jenkins.md) -- pipeline job

## Tips

### Cache the binary

For CI systems with persistent workspaces (not clean checkouts), skip the install step:

```shell
go-crap scan --fail-above --threshold 30 --exclude '.*_test\.go'
```

### Combine exclude patterns

Multiple `--exclude` flags stack:

```shell
go-crap scan \
  --exclude '.*_test\.go' \
  --exclude 'testdata/.*\.go' \
  --exclude '\.pb\.go$' \
  --exclude 'mock_'
```

### Threshold tuning

Start permissive, tighten over time:

```shell
# Phase 1: observe only (never fail)
go-crap scan --format json > crap-report.json

# Phase 2: enforce at a high threshold to see what fails
go-crap scan --fail-above --threshold 100

# Phase 3: tighten to production threshold
go-crap scan --fail-above --threshold 30

# Phase 4: strictest
go-crap scan --fail-above --threshold 15
```

### Debug logging in CI

```yaml
      - name: Debug scan
        run: go-crap scan --verbose --format json --exclude '.*_test\.go'
```

Use `--verbose` when diagnosing issues with module discovery, coverage parsing, or path matching in CI environments.

### Mutation report with CI

```yaml
      - name: Install gremlins
        run: go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
      - name: Run mutation testing
        run: gremlins unleash --output=gremlins-report.json
      - name: Run go-crap with mutation validation
        run: go-crap scan --mutation-report gremlins-report.json --fail-above --threshold 30
      - name: Generate detailed report
        run: go-crap scan --mutation-report gremlins-report.json --format json --detailed > crap-detailed.json
```

### Report parsing

The JSON output follows this schema:

```json
{
  "$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
  "version": "1.0.0",
  "entries": [
    {
      "file": "internal/pkg/foo.go",
      "package": "github.com/org/repo/internal/pkg",
      "function": "Foo",
      "line": 42,
      "cyclomatic": 8,
      "coverage": 0.25,
      "crap": 125.0,
      "coverage_untrusted": false,
      "mutation_score": 0.8,
      "effective_crap": 125.0
    }
  ]
}
```

When `--detailed` is used alongside `--mutation-report`, each entry with survived mutants includes a `mutation_details` array:

```json
{
  "mutation_details": [
    {
      "type": "CONDITIONALS_BOUNDARY",
      "mutator_name": "CB",
      "file": "internal/pkg/foo.go",
      "line": 50,
      "status": "LIVED",
      "original_text": "a < b",
      "replacement_text": "a >= b"
    }
  ]
}
```

Use `jq` to filter or summarize:

```shell
# Count functions exceeding threshold
go-crap scan --format json --threshold 30 | jq '[.entries[] | select(.crap > 30)] | length'

# Worst offenders
go-crap scan --format json --top 5 | jq '.entries[] | "\(.file):\(.line) \(.function) CRAP=\(.crap)"'
```
