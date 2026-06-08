package mutation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReport_valid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	data := `{
		"go_module": "github.com/example/test",
		"files": [
			{
				"file_name": "internal/score/score.go",
				"mutations": [
					{"type": "CONDITIONALS_BOUNDARY", "status": "LIVED", "line": 42},
					{"type": "ARITHMETIC", "status": "KILLED", "line": 43},
					{"type": "CONTROL_FLOW", "status": "NOT COVERED", "line": 50}
				]
			}
		],
		"mutants_killed": 1,
		"mutants_lived": 1,
		"mutants_not_covered": 1,
		"mutants_total": 3,
		"test_efficacy": 0.33
	}`
	err := os.WriteFile(path, []byte(data), 0644)
	require.NoError(t, err)

	report, err := ParseReport(path)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, "github.com/example/test", report.GoModule)
	assert.Equal(t, 1, report.MutantsKilled)
	assert.Equal(t, 1, report.MutantsLived)
	assert.Equal(t, 3, len(report.Mutants))

	assert.Equal(t, "internal/score/score.go", report.Mutants[0].File)
	assert.Equal(t, 42, report.Mutants[0].Line)
	assert.Equal(t, "CONDITIONALS_BOUNDARY", report.Mutants[0].Type)
	assert.Equal(t, StatusLived, report.Mutants[0].Status)

	assert.Equal(t, MutantStatus("KILLED"), report.Mutants[1].Status)
	assert.Equal(t, MutantStatus("NOT COVERED"), report.Mutants[2].Status)
}

func TestParseReport_empty_path(t *testing.T) {
	report, err := ParseReport("")
	assert.NoError(t, err)
	assert.Nil(t, report)
}

func TestParseReport_malformed_json(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	err := os.WriteFile(path, []byte("{not valid json"), 0644)
	require.NoError(t, err)

	report, err := ParseReport(path)
	assert.Error(t, err)
	assert.Nil(t, report)
}

func TestParseReport_nonexistent_file(t *testing.T) {
	report, err := ParseReport("/nonexistent/path/report.json")
	assert.Error(t, err)
	assert.Nil(t, report)
}

func TestParseReport_empty_mutants(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.json")
	data := `{"go_module": "test", "files": [], "mutants_killed": 0, "mutants_lived": 0, "mutants_not_covered": 0, "mutants_total": 0, "test_efficacy": 0}`
	err := os.WriteFile(path, []byte(data), 0644)
	require.NoError(t, err)

	report, err := ParseReport(path)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, "test", report.GoModule)
	assert.Equal(t, 0, len(report.Mutants))
}

func TestParseReport_multiple_files(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multi.json")
	data := `{
		"go_module": "test",
		"files": [
			{
				"file_name": "a.go",
				"mutations": [
					{"type": "T1", "status": "LIVED", "line": 10}
				]
			},
			{
				"file_name": "b.go",
				"mutations": [
					{"type": "T2", "status": "KILLED", "line": 20}
				]
			}
		],
		"mutants_killed": 1,
		"mutants_lived": 1,
		"mutants_not_covered": 0,
		"mutants_total": 2,
		"test_efficacy": 0.5
	}`
	err := os.WriteFile(path, []byte(data), 0644)
	require.NoError(t, err)

	report, err := ParseReport(path)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, 2, len(report.Mutants))
	assert.Equal(t, "a.go", report.Mutants[0].File)
	assert.Equal(t, 10, report.Mutants[0].Line)
	assert.Equal(t, "LIVED", string(report.Mutants[0].Status))
	assert.Equal(t, "b.go", report.Mutants[1].File)
	assert.Equal(t, 20, report.Mutants[1].Line)
	assert.Equal(t, "KILLED", string(report.Mutants[1].Status))
}

func TestMutantStatusConstants(t *testing.T) {
	assert.Equal(t, MutantStatus("LIVED"), StatusLived)
	assert.Equal(t, MutantStatus("KILLED"), StatusKilled)
	assert.Equal(t, MutantStatus("NOT COVERED"), StatusNotCover)
}
