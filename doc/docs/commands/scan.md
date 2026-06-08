# scan

The `scan` command is the main entry point. It analyzes Go modules, computes CRAP scores, and outputs the results.

## Usage

```shell
go-crap scan [path] [flags]
```

**Arguments:**

| Argument | Description | Default |
| - | - | - |
| `path` | Directory or package pattern to scan | `.` (current directory) |

## Flags

| Flag | Short | Description | Default |
| - | - | - | - |
| `--threshold` | `-t` | Score above which a function is marked as problematic | `30.0` |
| `--fail-above` | | Exit with code 1 if any function exceeds the threshold | `false` |
| `--format` | `-f` | Output format: `table`, `json`, `github`, `sarif`, or `pr-comment` | `table` |
| `--top` | | Show only the N worst offenders (0 = all) | `0` |
| `--min` | | Hide entries below this score | `0` |
| `--missing` | | Policy for functions without coverage: `pessimistic`, `optimistic`, or `skip` | `pessimistic` |
| `--exclude` | | Exclude files matching this regex pattern (repeatable). Use `.*` to match any path depth. e.g. `.*_test\.go` to exclude all test files, `pb/.*\.go` to exclude protobuf files | none |
| `--verbose` | | Enable verbose (debug-level) logging | `false` |
| `--output` | `-o` | Output file path (default: stdout) | stdout |

## Examples

### Scan all packages

```shell
go-crap scan
```

### Scan a project somewhere else

```shell
go-crap scan ~/go/src/github.com/padiazg/go-crap
```

### Show only the top 20 worst offenders

```shell
go-crap scan --top 20
```

### CI integration - fail on high CRAP scores

```shell
go-crap scan --fail-above --threshold 30 --format github
```

### Filter by minimum score

```shell
go-crap scan --min 10
```

### Exclude generated or test files

```shell
go-crap scan --exclude '.*_test\.go' --exclude 'testdata/.*\.go'
```

### Exclude protobuf and mock files at any depth

```shell
go-crap scan --exclude '\.pb\.go$' --exclude 'mock_'
```

### Machine-readable JSON output

```shell
go-crap scan --format json > report.json
```

### SARIF output

```shell
go-crap scan --format sarif > report.sarif
```

Outputs [SARIF 2.1.0](https://docs.oasis-open.org/sarif/sarif/v2.1.0/) compliant JSON for integration with code scanning tools, IDEs, and CI platforms that support SARIF.

### Pull request comment output

```shell
go-crap scan --format pr-comment > pr-comment.md
```

Generates a markdown table with status symbols, CRAP scores, complexity, coverage, and file locations — formatted for pasting into pull request comments.

### Write to file

```shell
go-crap scan --output report.json
go-crap scan -o report.json
```

Uses the `--output` / `-o` flag to write results to a file instead of stdout. Works with any format.

### Verbose / debug logging

```shell
go-crap scan --verbose
```

Enables debug-level logging to help diagnose issues with module discovery, coverage parsing, or path matching.
