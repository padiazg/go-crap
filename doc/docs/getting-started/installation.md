# Installation

## Prerequisites

- Go 1.22 or later

## Install via curl (Recommended)

Install the pre-built binary with version info stamped via ldflags. Prefer this over `go install` which rebuilds from source and loses version/commit/buildDate stamping.

```shell
curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh
```

Or install a specific version:

```shell
curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh -s -- -v v0.4.1
```

The binary is placed in `$GOPATH/bin` (or `$GOBIN` if set). Make sure that directory is in your `PATH`.

## Install via `go install`

```shell
go install github.com/padiazg/go-crap@latest
```

> **Note:** `go install` rebuilds the binary from source. The version will show as `v0.0.0 unknown unknown` because ldflags are not applied during `go install`. Use the curl installer above for release binaries with proper version info.

The binary is placed in `$GOPATH/bin` (or `$GOBIN` if set). Make sure that directory is in your `PATH`.

## Build from Source

```shell
git clone https://github.com/padiazg/go-crap.git
cd go-crap
make build
```

## Install via Homebrew

```shell
brew tap padiazg/go-crap
brew install go-crap
```

## Verify Installation

```shell
go-crap scan --help
```

This prints the command help text, confirming the binary is working.
