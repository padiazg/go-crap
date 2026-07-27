package report

import (
	"io"
	"path/filepath"

	"github.com/padiazg/go-crap/internal/scan"
)

// Formatter outputs CRAP entries in a specific format.
type Formatter interface {
	Format(entries *scan.Entries, opts FormatOptions) error
}

// FormatOptions configures report formatting behavior.
type FormatOptions struct {
	Writer    io.Writer
	Baseline  *Baseline
	Summary   *Summary
	BaseDir   string
	Threshold float64
	Detailed  bool
}

// Summary holds aggregate CRAP statistics across all functions.
type Summary struct {
	Combined         float64
	Average          float64
	TotalFuncs       int
	Exceeded         int
	BaselineCombined float64
	BaselineAverage  float64
	DeltaCombined    float64
	DeltaAverage     float64
}

// ComputeSummary computes aggregate CRAP stats from entries using the given threshold.
func ComputeSummary(entries *scan.Entries, threshold float64) Summary {
	if entries == nil {
		return Summary{}
	}
	list := entries.FullList
	if list == nil {
		list = entries.ForTable()
	}
	var combined float64
	exceeded := 0
	for _, e := range list {
		combined += e.EffectiveScore()
		if e.EffectiveScore() > threshold {
			exceeded++
		}
	}
	total := len(list)
	var avg float64
	if total > 0 {
		avg = combined / float64(total)
	}
	return Summary{
		Combined:   combined,
		Average:    avg,
		TotalFuncs: total,
		Exceeded:   exceeded,
	}
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
