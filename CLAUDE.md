# go-crap

CLI tool that calculates the CRAP score (cyclomatic complexity × coverage)
for Go projects. Inspired by cargo-crap (Rust).

## Skills

Read the Go skills before writing any code:

go-testgen
golang-benchmark
golang-cli
golang-code-style
golang-concurrency
golang-context
golang-continuous-integration
golang-data-structures
golang-database
golang-dependency-injection
golang-dependency-management
golang-design-patterns
golang-documentation
golang-error-handling
golang-linter
golang-modernize
golang-naming
golang-observability
golang-performance
golang-popular-libraries
golang-project-layout
golang-safety
golang-security
golang-stay-updated
golang-stretchr-testify
golang-structs-interfaces
golang-troubleshooting

Apply skills as style guide, patterns, and conventions.
They take precedence over default preferences.

## Test Generation

Use go-testgen for all test scaffolding. Do not write *_test.go by hand.

    # Check what needs tests in a package
    go-testgen report ./internal/<pkg>/...

    # Generate scaffolding
    go-testgen gen ./internal/<pkg> <FuncName>
    go-testgen gen ./internal/<pkg> Receiver.Method
    go-testgen gen ./internal/<pkg> Receiver.Method --mock-from pkg.Interface

The generated style (checkXxxFn closures, before field, tests []struct table)
is the canonical style for this project. Do not deviate from it.

If go-testgen is not installed:
    go install github.com/padiazg/go-testgen/cmd/go-testgen@latest

Use the `gen-test-cases` and `closure-check-tests` skills as appropriate.

## Useful Commands

    go build ./...                          # compile
    go test ./...                           # all tests
    go test -short ./...                    # skip integration tests
    go test -v ./internal/score/...         # a specific package
    go-testgen report ./internal/<pkg>/...  # check pending test coverage

## Architecture

Modules are independent and independently testable:
- internal/complexity  — AST walking (adapted from gocyclo, BSD-3)
- internal/coverage    — module discovery + coverage (adapted from test-finder)
- internal/merge       — join by (filepath, funcname) with double index
- internal/score       — CRAP formula + missing policy
- internal/report      — formatters (table, json, github)
- cmd/                 — CLI Cobra

## Critical Issue: path matching

coverage and complexity produce paths in different formats.
See internal/merge/index.go — NEVER canonicalize relative paths against CWD.
The test TestMerge_RelativePathsNotResolvedAgainstCWD enforces this invariant.

## Attribution

internal/complexity/* adapted from github.com/fzipp/gocyclo (BSD-3-Clause).
internal/coverage/* adapted from github.com/padiazg/test-finder (MIT).

# context-mode — MANDATORY routing rules

You have context-mode MCP tools available. These rules are NOT optional — they protect your context window from flooding. A single unrouted command can dump 56 KB into context and waste the entire session.

## BLOCKED commands — do NOT attempt these

### curl / wget — BLOCKED
Any Bash command containing `curl` or `wget` is intercepted and replaced with an error message. Do NOT retry.
Instead use:
- `ctx_fetch_and_index(url, source)` to fetch and index web pages
- `ctx_execute(language: "javascript", code: "const r = await fetch(...)")` to run HTTP calls in sandbox

### Inline HTTP — BLOCKED
Any Bash command containing `fetch('http`, `requests.get(`, `requests.post(`, `http.get(`, or `http.request(` is intercepted and replaced with an error message. Do NOT retry with Bash.
Instead use:
- `ctx_execute(language, code)` to run HTTP calls in sandbox — only stdout enters context

### WebFetch — BLOCKED
WebFetch calls are denied entirely. The URL is extracted and you are told to use `ctx_fetch_and_index` instead.
Instead use:
- `ctx_fetch_and_index(url, source)` then `ctx_search(queries)` to query the indexed content

## REDIRECTED tools — use sandbox equivalents

### Bash (>20 lines output)
Bash is ONLY for: `git`, `mkdir`, `rm`, `mv`, `cd`, `ls`, `npm install`, `pip install`, and other short-output commands.
For everything else, use:
- `ctx_batch_execute(commands, queries)` — run multiple commands + search in ONE call
- `ctx_execute(language: "shell", code: "...")` — run in sandbox, only stdout enters context

### Read (for analysis)
If you are reading a file to **Edit** it → Read is correct (Edit needs content in context).
If you are reading to **analyze, explore, or summarize** → use `ctx_execute_file(path, language, code)` instead. Only your printed summary enters context. The raw file content stays in the sandbox.

### Grep (large results)
Grep results can flood context. Use `ctx_execute(language: "shell", code: "grep ...")` to run searches in sandbox. Only your printed summary enters context.

## Tool selection hierarchy

1. **GATHER**: `ctx_batch_execute(commands, queries)` — Primary tool. Runs all commands, auto-indexes output, returns search results. ONE call replaces 30+ individual calls.
2. **FOLLOW-UP**: `ctx_search(queries: ["q1", "q2", ...])` — Query indexed content. Pass ALL questions as array in ONE call.
3. **PROCESSING**: `ctx_execute(language, code)` | `ctx_execute_file(path, language, code)` — Sandbox execution. Only stdout enters context.
4. **WEB**: `ctx_fetch_and_index(url, source)` then `ctx_search(queries)` — Fetch, chunk, index, query. Raw HTML never enters context.
5. **INDEX**: `ctx_index(content, source)` — Store content in FTS5 knowledge base for later search.

## Subagent routing

When spawning subagents (Agent/Task tool), the routing block is automatically injected into their prompt. Bash-type subagents are upgraded to general-purpose so they have access to MCP tools. You do NOT need to manually instruct subagents about context-mode.

## Output constraints

- Keep responses under 500 words.
- Write artifacts (code, configs, PRDs) to FILES — never return them as inline text. Return only: file path + 1-line description.
- When indexing content, use descriptive source labels so others can `ctx_search(source: "label")` later.

## ctx commands

| Command | Action |
|---------|--------|
| `ctx stats` | Call the `ctx_stats` MCP tool and display the full output verbatim |
| `ctx doctor` | Call the `ctx_doctor` MCP tool, run the returned shell command, display as checklist |
| `ctx upgrade` | Call the `ctx_upgrade` MCP tool, run the returned shell command, display as checklist |
