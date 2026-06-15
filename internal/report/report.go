package report

import (
	"io"
	"path/filepath"

	"github.com/padiazg/go-crap/internal/score"
)

type Formatter interface {
	Format(entries *score.EntryList, opts FormatOptions) error
}

type FormatOptions struct {
	Writer    io.Writer
	BaseDir   string
	Threshold float64
	Detailed  bool
}

// RelativizePath converts filePath to a path relative to baseDir.
// Returns the original path if relativization fails or baseDir is empty.
func RelativizePath(filePath, baseDir string) string {
	if baseDir == "" {
		return filePath
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return filePath
	}
	rel, err := filepath.Rel(absBase, filePath)
	if err != nil || rel == filePath {
		return filePath
	}
	return rel
}
