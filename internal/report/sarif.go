package report

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/padiazg/go-crap/internal/score"
)

type SARIFFormatter struct{}

func (f *SARIFFormatter) Format(entries *score.EntryList, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	results := make([]sarifResult, 0)
	for _, e := range entries.List {
		if e.CRAP > opts.Threshold {
			results = append(results, sarifResult{
				RuleID:  "crap/high-score",
				Level:   "warning",
				Message: sarifMessage{
					Text: formatMessage(e),
				},
				Locations: []sarifLocation{
					{
						PhysicalLocation: sarifPhysicalLocation{
							ArtifactLocation: sarifArtifactLocation{
								URI: relativizePath(e.File, opts.BaseDir),
							},
							Region: sarifRegion{
								StartLine: e.Line,
							},
						},
					},
				},
			})
		}
	}

	log := sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:             "go-crap",
						InformationURI:   "https://github.com/padiazg/go-crap",
						DefaultConfiguration: sarifDefaultConfiguration{
							Level: "warning",
						},
						Rules: []sarifRule{
							{
								ID: "crap/high-score",
								ShortDescription: sarifShortFullDescription{
									Text: "CRAP score exceeds threshold",
								},
								FullDescription: sarifShortFullDescription{
									Text: "The CRAP score (cyclomatic complexity × coverage) exceeds the threshold, indicating a function that is complex and/or poorly tested.",
								},
							},
						},
					},
				},
				Results: results,
			},
		},
	}

	enc := json.NewEncoder(opts.Writer)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}

func formatMessage(e score.CRAPEntry) string {
	name := e.FuncName
	if e.Receiver != "" {
		name = fmt.Sprintf("%s.%s", e.Receiver, e.FuncName)
	}
	return fmt.Sprintf("Function %s has CRAP score %.1f (cyclomatic complexity %d, coverage %.1f%%)",
		name, e.CRAP, e.Complexity, e.Coverage)
}

func relativizePath(path, baseDir string) string {
	if baseDir != "" {
		if absBase, err := filepath.Abs(baseDir); err == nil {
			if rel, err := filepath.Rel(absBase, path); err == nil && rel != path {
				path = rel
			}
		}
	}
	path = strings.ReplaceAll(path, `\`, `/`)
	return path
}

type sarifLog struct {
	Schema  string      `json:"$schema"`
	Version string      `json:"version"`
	Runs    []sarifRun  `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool    `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name             string          `json:"name"`
	InformationURI   string          `json:"informationUri"`
	DefaultConfiguration sarifDefaultConfiguration `json:"defaultConfiguration"`
	Rules            []sarifRule     `json:"rules"`
}

type sarifDefaultConfiguration struct {
	Level string `json:"level"`
}

type sarifRule struct {
	ID                 string              `json:"id"`
	ShortDescription   sarifShortFullDescription `json:"shortDescription"`
	FullDescription    sarifShortFullDescription `json:"fullDescription"`
}

type sarifShortFullDescription struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string           `json:"ruleId"`
	Level     string           `json:"level"`
	Message   sarifMessage     `json:"message"`
	Locations []sarifLocation  `json:"locations"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}
