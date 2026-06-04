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
| `--format` | `-f` | Output format: `table`, `json`, or `github` | `table` |
| `--top` | | Show only the N worst offenders (0 = all) | `0` |
| `--min` | | Hide entries below this score | `0` |
| `--missing` | | Policy for functions without coverage: `pessimistic`, `optimistic`, or `skip` | `pessimistic` |
| `--exclude` | | Exclude files matching this regex pattern (repeatable). Use `.*` to match any path depth. e.g. `.*_test\.go` to exclude all test files, `pb/.*\.go` to exclude protobuf files | none |

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
