# Understanding the Metrics Behind the CRAP Score

The CRAP score isn't a new, independent measurement — at its core it's a combination of two signals you probably already know: cyclomatic complexity and test coverage. go-crap adds a third signal, mutation testing, as an optional extra layer that checks whether the coverage number can actually be trusted. This page walks through what each one means on its own, what it tells you, and how they combine into the final number.

For the technical reference (formulas, CLI flags, CI integration), see [CRAP Score](https://padiazg.github.io/go-crap/concepts/crap-score/), [Mutation Testing](https://padiazg.github.io/go-crap/concepts/mutation-testing/), and [Missing Coverage Policy](https://padiazg.github.io/go-crap/concepts/missing-policy/).

## The two signals in the original CRAP formula

### 1. Cyclomatic complexity (CC)

**What it measures.** How many distinct paths a function can take. Every `if`, `for`, `case`, `&&`, `||` adds one more path.

**What it tells you on its own.** How hard the function is to reason about, and how many test cases you'd theoretically need to exercise every branch. A function with CC=1 has no decisions — a single possible path. A function with CC=15 has 15 possible flow combinations, and it's unlikely anyone tested all of them.

**What it doesn't tell you.** Whether those paths are actually tested. A function can be highly complex with 100% coverage, or highly complex with 0% coverage — complexity alone doesn't distinguish between the two.

### 2. Test coverage

**What it measures.** What percentage of a function's lines executed while running the test suite (`go test -cover`).

**What it tells you on its own.** Whether the code *ran* during tests. It's the most familiar metric, and also the easiest to misread.

**What it doesn't tell you.** Whether what ran was actually *verified*. A test can call a function and never make a meaningful assertion on the result — the line counts as "covered" even though the behavior was never checked. This is often called coverage without real assertions.

## How they combine into the CRAP score

```math
CRAP = CC² × (1 - coverage)³ + CC
```

This is the original formula, and it only uses the two signals above. In plain terms: **complexity gets punished hard when coverage is low, and that penalty mostly disappears when coverage is high.**

- High CC + high coverage → low CRAP (a complex but well-tested function isn't that risky)
- High CC + low coverage → very high CRAP (the worst case: hard-to-reason-about logic that nobody tested)
- Low CC → CRAP stays low regardless of coverage (a simple function isn't that risky even without tests)

## The optional third signal: mutation testing as a trust check

Mutation testing isn't part of the original CRAP formula — it's a layer go-crap adds on top, specifically to answer a question the formula can't: *is the coverage number even real?*

**What it measures.** The tool ([gremlins](https://github.com/go-gremlins/gremlins)) automatically alters small fragments of the source — for example, swapping `<` for `>=` — producing "mutants," then runs the test suite against each one.

- If a test fails → the mutant was **killed**: your tests caught the change. Good sign.
- If every test still passes → the mutant **survived**: your tests wouldn't have noticed the behavior changed. This exposes coverage that exists but protects nothing.

**What it tells you on its own.** Whether your coverage is *trustworthy* — whether the lines that ran were actually verified, not just executed. It doesn't say anything about complexity: a mutant can survive in a trivial one-line function just as easily as in a complex one.

**How it affects the score.** When a mutation report is available, go-crap computes a separate value, `EffectiveCRAP`: if any mutant survived inside a function, that function is recalculated *as if it had 0% coverage*, regardless of what `go test -cover` reported. Reported coverage doesn't count toward the score unless it's actually validated. This is a go-crap-specific safeguard, not something the original CRAP metric defines — you can use go-crap perfectly well with just CC and coverage, and add mutation testing later once you want a stronger guarantee that the coverage number means something.

## What weight the final score should carry

The CRAP score is not an absolute verdict on code quality — it's an indicator of **where to prioritize testing effort**. A high score doesn't mean "bad code"; it means "this function combines complexity with unverified behavior, so if something breaks here, you probably won't find out until production."

| CRAP range | What it suggests |
| - | - |
| 0 - 10 | Simple or well-tested function — low risk |
| 10 - 30 | Moderate complexity, worth a look |
| 30 - 50 | Clear candidate for refactoring or more tests |
| 50+ | High risk: complex logic with no reliable verification |

How much weight to give the score depends on what you plan to do with it:

- **As a triage tool** — sort functions by CRAP and start there when deciding what to test or refactor next, instead of guessing.
- **As a CI gate** (`--fail-above`) — block merges when new or changed code crosses a threshold, so complexity-without-tests doesn't creep in silently.
- **As a trend, not a single snapshot** — a stable or improving CRAP score over time matters more than any one number. A legacy function sitting at CRAP=60 that nobody touches is a different conversation than a new function shipped at CRAP=60 this week.

The score is most useful when it's paired with a decision: refactor, add tests, add mutation coverage, or explicitly accept the risk and move on. Without that follow-through, it's just a number — which is really the question worth asking before adopting it: not "is this score good or bad," but "what will we actually do when it's bad."
