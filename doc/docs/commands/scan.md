# scan

The `scan` command is the main entry point. It analyzes Go modules, computes CRAP scores, and outputs the results.

## Usage

```
go-crap scan [path] [flags]
```

**Arguments:**

| Argument | Description | Default |
|----------|-------------|---------|
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
| `--exclude` | | Exclude files matching this glob (repeatable) | none |

## Examples

### Scan all packages

```bash
go-crap scan
```

### Scan a project somewhere else

```bash
go-crap scan ~/go/src/github.com/padiazg/go-crap
```

### Show only the top 20 worst offenders

```bash
go-crap scan --top 20
```

### CI integration - fail on high CRAP scores

```bash
go-crap scan --fail-above --threshold 30 --format github
```

### Filter by minimum score

```bash
go-crap scan --min 10
```

### Exclude generated or test files

```bash
go-crap scan --exclude '*_test.go' --exclude '*/testdata/*'
```

### Machine-readable JSON output

```bash
go-crap scan --format json > report.json
```
