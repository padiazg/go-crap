package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

// SARIFFormatter outputs CRAP entries as SARIF JSON.
type SARIFFormatter struct{}

func (f *SARIFFormatter) Format(entries *scan.Entries, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	results := make([]sarifResult, 0)
	for _, e := range entries.List {
		effectiveCRAP := e.EffectiveScore()

		if effectiveCRAP > opts.Threshold {
			results = append(results, sarifResult{
				RuleID: "crap/high-score",
				Level:  "warning",
				Message: sarifMessage{
					Text: formatMessage(e, effectiveCRAP, opts.Detailed),
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

		if e.CoverageUntrusted {
			msg := fmt.Sprintf("Coverage not reliable for %s (mutation score: %.1f%%)", e.FuncName, e.MutationScore*100)
			msg += formatMutantDetails(opts.Detailed, e.MutationDetails)
			results = append(results, sarifResult{
				RuleID: "go-crap/coverage-untrusted",
				Level:  "warning",
				Message: sarifMessage{
					Text: msg,
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
						Name:           "go-crap",
						InformationURI: "https://github.com/padiazg/go-crap",
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

func formatMessage(e score.CRAPEntry, effectiveCRAP float64, detailed bool) string {
	name := e.FuncName
	if e.Receiver != "" {
		name = fmt.Sprintf("%s.%s", e.Receiver, e.FuncName)
	}
	msg := fmt.Sprintf("Function %s has CRAP score %.1f (cyclomatic complexity %d, coverage %.1f%%)",
		name, effectiveCRAP, e.Complexity, e.Coverage)
	if e.CoverageUntrusted {
		msg += fmt.Sprintf(" [coverage not reliable (mutation score: %.1f%%)]", e.MutationScore*100)
		msg += formatMutantDetails(detailed, e.MutationDetails)
	}
	return msg
}

func formatMutantDetails(detailed bool, details []score.MutationDetail) string {
	if !detailed || len(details) == 0 {
		return ""
	}
	var msg strings.Builder
	msg.WriteString(" survived mutations:")
	for _, md := range details {
		fmt.Fprintf(&msg, " %s@L%d", md.MutantType, md.Line)
		if md.OriginalText != "" && md.ReplacementText != "" {
			fmt.Fprintf(&msg, " (%q → %q)", md.OriginalText, md.ReplacementText)
		}
	}
	return msg.String()
}

func relativizePath(path, baseDir string) string {
	path = RelativizePath(path, baseDir)
	path = strings.ReplaceAll(path, `\`, `/`)
	return path
}

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Results []sarifResult `json:"results"`
	Tool    sarifTool     `json:"tool"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name                 string                    `json:"name"`
	InformationURI       string                    `json:"informationUri"`
	DefaultConfiguration sarifDefaultConfiguration `json:"defaultConfiguration"`
	Rules                []sarifRule               `json:"rules"`
}

type sarifDefaultConfiguration struct {
	Level string `json:"level"`
}

type sarifRule struct {
	ID               string                    `json:"id"`
	ShortDescription sarifShortFullDescription `json:"shortDescription"`
	FullDescription  sarifShortFullDescription `json:"fullDescription"`
}

type sarifShortFullDescription struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
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
