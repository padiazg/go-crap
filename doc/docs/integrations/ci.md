# CI Workflows

Integrate go-crap into your continuous integration pipeline to enforce CRAP score thresholds across all pull requests.

## GitHub Actions

### Fail on high CRAP scores

```yaml
name: crap
on:
  push:
    branches: [main, master]
  pull_request:

jobs:
  crap:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true
      - name: Install go-crap
        run: go install github.com/padiazg/go-crap@latest
      - name: Run go-crap
        run: go-crap scan --fail-above --threshold 30 --exclude '.*_test\.go' --exclude 'testdata/.*\.go' --exclude '\.pb\.go$'
```

### PR annotations with JSON report

```yaml
      - name: Install go-crap
        run: go install github.com/padiazg/go-crap@latest
      - name: Run go-crap with annotations
        run: go-crap scan --format github --threshold 30 --exclude '.*_test\.go'
      - name: Generate JSON report
        run: go-crap scan --format json > crap-report.json
      - uses: actions/upload-artifact@v4
        with:
          name: crap-report
          path: crap-report.json
```

### Matrix builds across Go versions

```yaml
name: crap
on: [push, pull_request]

jobs:
  crap:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.22', '1.23', '1.24']
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
          cache: true
      - name: Install go-crap
        run: go install github.com/padiazg/go-crap@latest
      - name: Run go-crap
        run: go-crap scan --fail-above --threshold 30 --exclude '.*_test\.go'
```

## Gitea CI

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

### Gitea Actions annotations

Gitea Actions supports GitHub-style `::warning` annotations:

```yaml
      - name: Run go-crap with annotations
        run: go-crap scan --format github --threshold 30 --exclude '.*_test\.go'
```

## GitLab CI

`.gitlab-ci.yml`:

```yaml
stages:
  - test
  - quality

crap:
  stage: quality
  image: golang:1.23
  before_script:
    - go install github.com/padiazg/go-crap@latest
  script:
    - go-crap scan --fail-above --threshold 30 --exclude '.*_test\.go' --exclude 'testdata/.*\.go'
  allow_failure: false
```

### SARIF report with code scanning

```yaml
      - name: Run go-crap with SARIF output
        run: go-crap scan --format sarif --threshold 30 --exclude '.*_test\.go' > report.sarif
      - name: Upload SARIF report
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: report.sarif
```

SARIF output is compatible with GitHub Advanced Security code scanning, Azure DevOps, and other tools that consume SARIF 2.1.0 reports.

### PR comment with CRAP report

```yaml
      - name: Run go-crap for PR comment
        run: go-crap scan --format pr-comment --threshold 30 --exclude '.*_test\.go' --output pr-comment.md
      - name: Comment on PR
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const comment = fs.readFileSync('pr-comment.md', 'utf8');
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: comment
            });
```

The `pr-comment` format generates a markdown table suitable for pull request comments, showing status symbols, CRAP scores, complexity, coverage, function names, and file locations.

### JSON report as CI artifact

```yaml
crap:
  stage: quality
  image: golang:1.23
  before_script:
    - go install github.com/padiazg/go-crap@latest
  script:
    - go-crap scan --format json > crap-report.json --exclude '.*_test\.go'
  artifacts:
    paths:
      - crap-report.json
    expire_in: 30 days
  allow_failure: true
```

## Azure DevOps

`.azure-pipelines.yml`:

```yaml
trigger:
  - main
  - master
pr:
  - main
  - master

pool:
  vmImage: 'ubuntu-latest'

steps:
  - task: GoTool@1
    inputs:
      version: '1.23'

  - script: |
      go install github.com/padiazg/go-crap@latest
      go-crap scan --fail-above --threshold 30 --exclude '.*_test\.go' --exclude 'testdata/.*\.go'
    displayName: 'Run go-crap'
```

### Upload JSON report as artifact

```yaml
  - script: |
      go install github.com/padiazg/go-crap@latest
      go-crap scan --format json > $(Build.ArtifactStagingDirectory)/crap-report.json --exclude '.*_test\.go'
    displayName: 'Generate CRAP report'

  - task: PublishBuildArtifacts@1
    inputs:
      pathToPublish: '$(Build.ArtifactStagingDirectory)'
      artifactName: 'crap-report'
      publishLocation: 'Container'
```

## CircleCI

`.circleci/config.yml`:

```yaml
version: 2.1

jobs:
  crap:
    docker:
      - image: cimg/go:1.23
    steps:
      - checkout
      - run:
          name: Install go-crap
          command: go install github.com/padiazg/go-crap@latest
      - run:
          name: Run go-crap
          command: go-crap scan --fail-above --threshold 30 --exclude '.*_test\.go' --exclude 'testdata/.*\.go'

workflows:
  quality:
    jobs:
      - crap
```

## Jenkins (Pipeline)

`Jenkinsfile`:

```groovy
pipeline {
    agent any

    environment {
        GO_VERSION = '1.23'
    }

    stages {
        stage('Install go-crap') {
            steps {
                sh "go install github.com/padiazg/go-crap@latest"
            }
        }
        stage('Run go-crap') {
            steps {
                sh "go-crap scan --fail-above --threshold 30 --exclude '.*_test\\.go' --exclude 'testdata/.*\\.go'"
            }
        }
    }
}
```

## Tips

### Cache the binary

For CI systems with persistent workspaces (not clean checkouts), skip the install step:

```shell
go-crap scan --fail-above --threshold 30 --exclude '.*_test\.go'
```

### Combine exclude patterns

Multiple `--exclude` flags stack:

```shell
go-crap scan \
  --exclude '.*_test\.go' \
  --exclude 'testdata/.*\.go' \
  --exclude '\.pb\.go$' \
  --exclude 'mock_'
```

### Threshold tuning

Start permissive, tighten over time:

```shell
# Phase 1: observe only (never fail)
go-crap scan --format json > crap-report.json

# Phase 2: enforce at a high threshold to see what fails
go-crap scan --fail-above --threshold 100

# Phase 3: tighten to production threshold
go-crap scan --fail-above --threshold 30

# Phase 4: strictest
go-crap scan --fail-above --threshold 15
```

### Debug logging in CI

```yaml
      - name: Debug scan
        run: go-crap scan --verbose --format json --exclude '.*_test\.go'
```

Use `--verbose` when diagnosing issues with module discovery, coverage parsing, or path matching in CI environments.

### Mutation report with CI

```yaml
      - name: Install gremlins
        run: go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
      - name: Run mutation testing
        run: gremlins unleash --output=gremlins-report.json
      - name: Run go-crap with mutation validation
        run: go-crap scan --mutation-report gremlins-report.json --fail-above --threshold 30
      - name: Generate detailed report
        run: go-crap scan --mutation-report gremlins-report.json --format json --detailed > crap-detailed.json
```

### Report parsing

The JSON output follows this schema:

```json
{
  "$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
  "version": "1.0.0",
  "entries": [
    {
      "file": "internal/pkg/foo.go",
      "package": "github.com/org/repo/internal/pkg",
      "function": "Foo",
      "line": 42,
      "cyclomatic": 8,
      "coverage": 0.25,
      "crap": 125.0,
      "coverage_untrusted": false,
      "mutation_score": 0.8,
      "effective_crap": 125.0
    }
  ]
}
```

When `--detailed` is used alongside `--mutation-report`, each entry with survived mutants includes a `mutation_details` array:

```json
{
  "mutation_details": [
    {
      "type": "CONDITIONALS_BOUNDARY",
      "mutator_name": "CB",
      "file": "internal/pkg/foo.go",
      "line": 50,
      "status": "LIVED",
      "original_text": "a < b",
      "replacement_text": "a >= b"
    }
  ]
}
```

Use `jq` to filter or summarize:

```shell
# Count functions exceeding threshold
go-crap scan --format json --threshold 30 | jq '[.entries[] | select(.crap > 30)] | length'

# Worst offenders
go-crap scan --format json --top 5 | jq '.entries[] | "\(.file):\(.line) \(.function) CRAP=\(.crap)"'
```
