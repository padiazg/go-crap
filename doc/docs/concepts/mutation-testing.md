## Mutation Testing

Mutation testing evaluates the quality of your tests by injecting small, deliberate changes (called **mutants**) into your source code and running tests to see if they catch them.

A mutant that gets caught by tests is **killed** — meaning your test caught the change. A mutant that survives means your tests have a blind spot: the code path was never meaningfully asserted.

Coverage alone can lie. A function with 100% line coverage can still have untested logical paths. Mutation testing catches that gap.

## Gremlins

[Gremlins](https://gremlins.dev/latest/) is a mutation testing tool for Go that go-crap integrates with via JSON reports.

### Installation

```shell
go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
```

### Generating a report

```shell
gremlins unleash --output=gremlins-report.json
```

Gremlins supports multiple mutators: conditional boundaries, increments, logical operators, function calls, and more. See the [Gremlins documentation](https://gremlins.dev/latest/) for the full list and configuration options.

## Mutation reports with go-crap

Use the `--mutation-report` flag to pass a gremlins JSON report to go-crap. go-crap matches mutants to functions by file and line range, then:

1. Counts **killed** vs **lived** mutants within each function's line range
2. If any mutant **lived** → coverage is marked **untrusted** → CRAP recalculated assuming 0% coverage
3. Computes `mutation_score` = `killed / (killed + lived)`

```shell
go-crap scan --mutation-report gremlins-report.json
```

Use `--detailed` alongside `--mutation-report` to include per-mutant details (type, line, original/replacement code):

```shell
go-crap scan --mutation-report gremlins-report.json --format json --detailed
```

### How mutation data surfaces in each format

| Format | Mutation indicator |
|--------|-------------------|
| `table` | ⚠ flag next to coverage percentage |
| `json` | `mutation_score`, `coverage_untrusted`, `mutation_details` array |
| `sarif` | `coverage-untrusted` result; survived mutations with code diffs appended to warning messages |
| `pr-comment` | "Unreliable Coverage" section + "Survived Mutants" column with inline code snippets |

## CI integration

### GitHub Actions

```yaml
name: mutation
on: [push, pull_request]

jobs:
  mutation:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true
      - name: Install gremlins
        run: go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
      - name: Install go-crap
        run: go install github.com/padiazg/go-crap@latest
      - name: Run mutation testing
        run: gremlins unleash --output=gremlins-report.json
      - name: Run go-crap with mutation validation
        run: go-crap scan --mutation-report gremlins-report.json --fail-above --threshold 30
```

See [CI Workflows](../integrations/ci.md) for platform-specific examples with SARIF, PR comments, and matrix builds.

## Interpreting results

### `effective_crap` vs `crap`

When no survived mutants exist, `effective_crap` equals `crap`. When mutants survived, `effective_crap` is recalculated assuming 0% coverage — reflecting the true risk of untested logic.

### Finding survived mutants

```shell
# Survived mutants per function
go-crap scan --mutation-report gremlins-report.json --format json --detailed | \
  jq '.entries[] | select(.mutation_details != null) | {file, function, mutation_details}'

# Summary of mutation scores
go-crap scan --mutation-report gremlins-report.json --format json | \
  jq '[.entries[] | select(.coverage_untrusted == true)] | length'
```

A `mutation_score` of 1.0 means all mutants were killed. A score near 0 means most survived — coverage in that function is unreliable.

### Gremlins report format

go-crap expects the JSON structure produced by `gremlins unleash`:

```json
{
  "go_module": "github.com/org/repo",
  "files": [
    {
      "file_name": "internal/pkg/foo.go",
      "mutations": [
        {
          "type": "CONDITIONALS_BOUNDARY",
          "mutator": "CB",
          "file": "internal/pkg/foo.go",
          "line": 42,
          "status": "LIVED",
          "original_code": "a < b",
          "replacement_code": "a >= b"
        }
      ]
    }
  ],
  "mutants_killed": 10,
  "mutants_lived": 2,
  "mutants_not_covered": 0,
  "mutants_total": 12,
  "test_efficacy": 0.833
}
```

Mutants are matched to functions by file path and line range. A mutant within a function's start-to-end line range is attributed to that function.

## Future compatibility

Other mutation testing tools like [ooze](https://github.com/gtramontina/ooze) could be supported in the future. Current support is gremlins-only since other alternatives are unmaintained.
