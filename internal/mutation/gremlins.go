package mutation

import (
	"encoding/json"
	"os"
)

type MutantStatus string

const (
	StatusLived    MutantStatus = "LIVED"
	StatusKilled   MutantStatus = "KILLED"
	StatusNotCover MutantStatus = "NOT COVERED"
)

type Mutant struct {
	Status          MutantStatus `json:"status"`
	File            string       `json:"-"`
	MutatorName     string       `json:"mutator,omitempty"`
	OriginalCode    string       `json:"original_code,omitempty"`
	ReplacementCode string       `json:"replacement_code,omitempty"`
	Type            string       `json:"type"`
	Line            int          `json:"line"`
}

type FileMutations struct {
	FileName  string   `json:"file_name"`
	Mutations []Mutant `json:"mutations"`
}

type Report struct {
	GoModule        string          `json:"go_module"`
	Files           []FileMutations `json:"files"`
	Mutants         []Mutant        `json:"-"`
	MutantsKilled   int             `json:"mutants_killed"`
	MutantsLived    int             `json:"mutants_lived"`
	MutantsNotCover int             `json:"mutants_not_covered"`
	MutantsTotal    int             `json:"mutants_total"`
	TestEfficacy    float64         `json:"test_efficacy"`
}

func ParseReport(path string) (*Report, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}

	// Flatten nested mutations into a single list
	mutants := make([]Mutant, 0)
	for _, fm := range report.Files {
		for i := range fm.Mutations {
			fm.Mutations[i].File = fm.FileName
			mutants = append(mutants, fm.Mutations[i])
		}
	}
	report.Mutants = mutants

	return &report, nil
}
