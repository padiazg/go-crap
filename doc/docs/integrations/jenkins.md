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
                sh "curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh"
            }
        }
        stage('Run go-crap') {
            steps {
                sh "go-crap scan --fail-above --threshold 30 --exclude 'testdata/.*\\.go'"
            }
        }
    }
}
```
