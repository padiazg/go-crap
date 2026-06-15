# CircleCI

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
