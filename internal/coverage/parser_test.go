package coverage

import (
	"strings"
	"testing"
)

func TestParseCoverOutput(t *testing.T) {
	input := `github.com/padiazg/go-crap/internal/merge/merge.go:45: Merge 87.5%
github.com/padiazg/go-crap/internal/merge/merge.go:12: buildIndex 45.2%
total: (statements) 82.3%
`
	functions, err := parseCoverOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(functions))
	}
	if functions[0].Name != "Merge" {
		t.Errorf("expected Merge, got %s", functions[0].Name)
	}
	if functions[0].Coverage != 87.5 {
		t.Errorf("expected 87.5 coverage, got %f", functions[0].Coverage)
	}
	if functions[0].Line != 45 {
		t.Errorf("expected line 45, got %d", functions[0].Line)
	}
}

func TestParseCoverOutput_SkipsTotal(t *testing.T) {
	input := `total: (statements) 82.3%
`
	functions, err := parseCoverOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) != 0 {
		t.Errorf("expected 0 functions, got %d", len(functions))
	}
}

func TestParseCoverBytes(t *testing.T) {
	input := []byte(`github.com/padiazg/go-crap/internal/merge/merge.go:45: Merge 87.5%
`)
	functions, err := ParseCoverBytes(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(functions))
	}
}
