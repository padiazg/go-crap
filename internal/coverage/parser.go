package coverage

import (
	"bytes"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type FunctionCoverage struct {
	File     string
	Package  string
	Name     string
	Line     int
	Coverage float64
}

type ModuleCoverage struct {
	Error      error
	Dir        string
	ModulePath string
	Functions  []FunctionCoverage
}

var coverLineRegex = regexp.MustCompile(`^([^:]+):(\d+):\s+(\S+)\s+([\d.]+)%`)

func parseCoverOutput(r io.Reader) ([]FunctionCoverage, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var results []FunctionCoverage
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total:") {
			continue
		}
		matches := coverLineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		lineNum, err := strconv.Atoi(matches[2])
		if err != nil {
			lineNum = 0
		}
		coverage, err := strconv.ParseFloat(matches[4], 64)
		if err != nil {
			coverage = 0
		}
		results = append(results, FunctionCoverage{
			File:     matches[1],
			Line:     lineNum,
			Name:     matches[3],
			Coverage: coverage,
		})
	}
	return results, nil
}

func ParseCoverBytes(data []byte) ([]FunctionCoverage, error) {
	return parseCoverOutput(bytes.NewReader(data))
}
