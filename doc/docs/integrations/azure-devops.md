# Azure DevOps

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

## Upload JSON report as artifact

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
