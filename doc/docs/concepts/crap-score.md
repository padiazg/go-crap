# CRAP Score

CRAP stands for **C**yclomatic **R**eadability **A**nd **P**redictability. It measures how expensive a function is to test.

## The Formula

$$
CRAP\left(CC, coverage\right) = CC^2 * \left(1 - \frac{coverage}{100} \right)^3 + CC
$$

Where:

- $CC$ - cyclomatic complexity (number of independent paths)
- $coverage$ - test coverage as a percentage (0-100)

## Interpretation

| CRAP Range | Meaning |
| - | - |
| 0 - 10 | Well-tested, simple function |
| 10 - 30 | Moderate complexity, should be tested |
| 30 - 50 | High CRAP - refactoring or more tests needed |
| 50+ | Critical - likely hard to test, complex logic |

## How It Works

### Cyclomatic Complexity

Computed by walking the AST and counting decision points:

- `if`, `else if`, `for`, `case`, `&&`, `||`, `?`

Each decision point adds 1 to the base complexity of 1.

### Coverage

Obtained by running `go test -coverprofile=...` and reading the coverage profile. Coverage is expressed as a percentage of lines executed within each function.

### Example

A function with CC=5 and 75% coverage:

$$
\begin{align}
CRAP &= 5^2 * (1 - 0.75)^3 + 5  \\
     &= 25 * 0.015625 + 5  \\
     &= 0.39 + 5  \\
     &= 5.39  \\
\end{align}
$$

The same function with 0% coverage:

$$
\begin{align}
CRAP &= 5^2 * (1 - 0)^3 + 5 \\
     &= 25 * 1 + 5 \\
     &= 30.0 \\
\end{align}
$$

As coverage drops, the CRAP score rises - poorly tested complex functions score highest.

## Reference

The original CRAP score was introduced by Adam Tornelius. go-crap uses the same formula as the [cargo-crap](https://github.com/Boehs/cargo-crap) Rust implementation.
