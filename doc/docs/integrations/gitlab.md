# GitLab CI

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

## JSON report as CI artifact

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
