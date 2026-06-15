# Jenkins (Pipeline)

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
