# Installation

## Prerequisites

- Go 1.22 or later

## Install via `go install`

```shell
go install github.com/padiazg/go-crap@latest
```

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
