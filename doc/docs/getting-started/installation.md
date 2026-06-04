# Installation

## Prerequisites

- Go 1.22 or later

## Install via `go install`

```bash
go install github.com/padiazg/go-crap@latest
```

The binary is placed in `$GOPATH/bin` (or `$GOBIN` if set). Make sure that directory is in your `PATH`.

## Build from Source

```bash
git clone https://github.com/padiazg/go-crap.git
cd go-crap
go build -o go-crap .
```

## Install via Homebrew

```bash
brew tap padiazg/go-crap
brew install go-crap
```

## Verify Installation

```bash
go-crap scan --help
```

This prints the command help text, confirming the binary is working.
