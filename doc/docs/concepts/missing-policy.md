# Missing Coverage Policy

When go-crap finds a function that has no coverage data at all, it needs a strategy. The `--missing` flag controls this behavior.

> **Note:** "no coverage data" means the function does not appear in any coverage profile — typically because the package has no test file, or the function is in a file excluded from the test build. This is different from a function that appears in a coverage profile with 0% coverage (all statements present but none executed). Functions with known 0% coverage are always reported with their actual score, regardless of the `--missing` policy.

## Options

| Policy | Flag | Effect |
|--------|------|--------|
| **pessimistic** | `--missing pessimistic` (default) | Assume 0% coverage x maximum CRAP score |
| **optimistic** | `--missing optimistic` | Assume 100% coverage x minimum CRAP score |
| **skip** | `--missing skip` | Exclude the function from results |

## Pessimistic (default)

Assumes functions without coverage have 0% test coverage. This gives the highest possible CRAP score and surfaces them as top issues.

```bash
go-crap scan --missing pessimistic
```

A function with CC=8 and 0% coverage:

$$
CRAP = 8^2 x (1 - 0)^3 + 8 = 64 + 8 = 72.00
$$

## Optimistic

Assumes functions without coverage have 100% coverage. This gives the minimum possible CRAP score - useful for surveys where you want to see the "best case" picture.

```bash
go-crap scan --missing optimistic
```

A function with CC=8 and 100% assumed coverage:

$$
CRAP = 8^2 * \left(1 - 1\right)^3 + 8 = 0 + 8 = 8.00
$$

## Skip

Excludes functions without coverage from the output entirely. Useful for ignoring untested functions and focusing only on those that have coverage data.

```bash
go-crap scan --missing skip
```

Functions without coverage appear in the output with `Skipped: true` and their CRAP score equals their complexity.

## When to Use Each

- **pessimistic** - CI checks, code reviews, finding the worst issues first
- **optimistic** - rough estimation, best-case scenario analysis
- **skip** - auditing covered code only, ignoring untested functions
