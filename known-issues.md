# Known Issues

Known issues and workarounds for go-crap.

---

## CI Workflow Go Version Mismatch

**Symptom:** `go: no such tool "covdata"` error during coverage scan, causing all functions to report 0% coverage.

```
time=... level=DEBUG msg="coverage scan: module error" error="go test: exit status 1\n# github.com/padiazg/go-crap\ngo: no such tool \"covdata\"\n"
```

**Cause:** The `release` workflow uses `go-version: '1.21'` while `go.mod` declares `go 1.26.2`. The mismatched Go version in CI causes the covdata tool to be incompatible with the module's declared Go version.

**Affected workflows:** `.github/workflows/release.yml`

**Fix:** Match the `go-version` in all CI workflows to the `go` directive in `go.mod`:

```yaml
# .github/workflows/release.yml
- uses: actions/setup-go@v5
  with:
    go-version: '1.26.2'  # must match go.mod
```

**Reference:** [Issue #6](https://github.com/padiazg/go-crap/issues/6)
