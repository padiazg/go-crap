package packages

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// GoListRunner runs `go list -json -e` and returns stdout bytes.
type GoListRunner interface {
	List(ctx context.Context, patterns []string) ([]byte, error)
}

// goListRunner is the default real implementation.
type goListRunner struct {
	dir string // working directory
	env []string // environment
}

// NewGoListRunner returns a GoListRunner with defaults:
//
//	dir = "."
//	env = os.Environ() + "GO111MODULE=on"
func NewGoListRunner() GoListRunner {
	return &goListRunner{
		dir: ".",
		env: append(os.Environ(), "GO111MODULE=on"),
	}
}

// NewGoListRunnerWithDir returns a runner that uses the given working directory.
func NewGoListRunnerWithDir(dir string) GoListRunner {
	r := NewGoListRunner().(*goListRunner)
	r.dir = dir
	return r
}

// WithEnv sets the environment for the runner.
// If called multiple times, the last call wins.
func WithEnv(env []string) GoListRunner {
	r := NewGoListRunner().(*goListRunner)
	r.env = env
	return r
}

func (r *goListRunner) List(ctx context.Context, patterns []string) ([]byte, error) {
	args := []string{"list", "-e", "-json"}
	args = append(args, patterns...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = r.dir
	cmd.Env = r.env
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	err := cmd.Run()
	if err != nil && stdout.Len() == 0 {
		return nil, err
	}

	return stdout.Bytes(), nil
}
