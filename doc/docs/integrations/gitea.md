# Gitea CI

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
        uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true
      - name: Install go-crap
        run: go install github.com/padiazg/go-crap@latest
      - name: Run go-crap
        run: go-crap scan --fail-above --threshold 30 --exclude '.*_test\.go' --exclude 'testdata/.*\.go'
```

## Annotations

Gitea Actions supports GitHub-style `::warning` annotations:

```yaml
      - name: Run go-crap with annotations
        run: go-crap scan --format github --threshold 30 --exclude '.*_test\.go'
```
