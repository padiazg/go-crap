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
      curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh
      go-crap scan --fail-above --threshold 30 --exclude '.*_test\.go' --exclude 'testdata/.*\.go'
    displayName: 'Run go-crap'
```

## Upload JSON report as artifact

```yaml
  - script: |
      curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh
      go-crap scan --format json > $(Build.ArtifactStagingDirectory)/crap-report.json --exclude '.*_test\.go'
    displayName: 'Generate CRAP report'

  - task: PublishBuildArtifacts@1
    inputs:
      pathToPublish: '$(Build.ArtifactStagingDirectory)'
      artifactName: 'crap-report'
      publishLocation: 'Container'
```
