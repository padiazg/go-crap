# Gitea CI

Two ways to run go-crap in Gitea Actions:

1. **Install the binary** with `install.sh` — no extra requirements
2. **Use the [go-crap action](https://github.com/marketplace/actions/go-crap)** from the GitHub Marketplace — no install step, but the worker needs Docker

## Use the go-crap action

Gitea Actions runs GitHub-Actions-compatible workflows, so the [go-crap action](https://github.com/padiazg/go-crap-action) works unchanged — unqualified `uses: owner/repo@ref` steps download from GitHub by default:

```yaml
---
name: crap
on: [push, pull_request]

jobs:
  crap:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v8
      - uses: padiazg/go-crap-action@v0.5.1
        with:
          args: scan --fail-above --threshold 30
```

Requirements:

- Gitea Actions enabled on the instance and repository, with a registered runner
- Runner can reach `github.com` (action download) and `ghcr.io` (image pull)
- Docker available on the runner — the action is a composite action that runs `docker run`

Intranet instances that cannot reach GitHub: mirror the action repo on your own instance and reference it with an absolute URL (or set `[actions].DEFAULT_ACTIONS_URL = "self"`):

```yaml
      - uses: https://git.example.com/your-org/go-crap-action@v0.5.1
        with:
          args: scan --fail-above --threshold 30
```

## Install the binary

Standard `.gitea/workflows/ci.yml` or `cicd.yml`:

```yaml
---
name: crap
on: [push, pull_request]

jobs:
  crap:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v8
      - name: Setup Go
        uses: actions/setup-go@v7
        with:
          go-version: '1.23'
          cache: true
      - name: Install go-crap
        run: curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh
      - name: Run go-crap
        run: go-crap scan --fail-above --threshold 30 --exclude 'testdata/.*\.go'
```

## Annotations

Gitea Actions supports GitHub-style `::warning` annotations:

```yaml
      - name: Run go-crap with annotations
        run: go-crap scan --format github --threshold 30
```
