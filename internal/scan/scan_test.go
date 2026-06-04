package scan

import (
	"testing"

	"github.com/padiazg/go-crap/internal/score"
	"github.com/stretchr/testify/assert"
)

type applyFiltersFn func(*testing.T, []score.CRAPEntry)

var (
	checkapplyFilters = func(fns ...applyFiltersFn) []applyFiltersFn { return fns }
	dataset           = score.EntryList{List: []score.CRAPEntry{
		{CRAP: 20.00, Coverage: 4, FuncName: "walkForModules", Package: "internal/coverage", File: "scanner.go", Line: 74},
		{CRAP: 56.00, Coverage: 7, FuncName: "Scan", Package: "internal/coverage", File: "scanner.go", Line: 21},
		{CRAP: 110.00, Coverage: 10, FuncName: "runScan", Package: "cmd", File: "scan.go", Line: 55},
		{CRAP: 42.00, Coverage: 6, FuncName: "applyFilters", Package: "cmd", File: "scan.go", Line: 135},
		{CRAP: 30.00, Coverage: 5, FuncName: "Merge", Package: "internal/merge", File: "merge.go", Line: 65},
		{CRAP: 30.00, Coverage: 5, FuncName: "scanModule", Package: "internal/coverage", File: "scanner.go", Line: 89},
		{CRAP: 20.00, Coverage: 4, FuncName: "filterByExclude", Package: "internal/coverage", File: "scanner.go", Line: 157},
		{CRAP: 20.00, Coverage: 4, FuncName: "parseMissingPolicy", Package: "cmd", File: "scan.go", Line: 122},
		{CRAP: 20.00, Coverage: 4, FuncName: "resolveFormatter", Package: "cmd", File: "scan.go", Line: 154},
		{CRAP: 20.00, Coverage: 4, FuncName: "readModulePath", Package: "internal/coverage", File: "scanner.go", Line: 121},
	}}
)

func Test_applyFilters(t *testing.T) {
	checkLen := func(count int) applyFiltersFn {
		return func(t *testing.T, c []score.CRAPEntry) {
			t.Helper()
			assert.Equal(t, count, len(c))
		}
	}

	checkValue := func(pos int, crap float64, name string) applyFiltersFn {
		return func(t *testing.T, c []score.CRAPEntry) {
			t.Helper()
			assert.Equal(t, crap, c[pos].CRAP)
			assert.Equal(t, name, c[pos].FuncName)

		}
	}

	tests := []struct {
		name    string
		entries score.EntryList
		top     int
		min     float64
		checks  []applyFiltersFn
	}{
		{
			name:    "full-list",
			entries: dataset,
			checks: checkapplyFilters(
				checkLen(10),
				checkValue(0, 110.00, "runScan"),
				checkValue(9, 20.00, "readModulePath"),
			),
		},
		{
			name:    "min-30",
			entries: dataset,
			min:     30.00,
			checks: checkapplyFilters(
				checkLen(5),
				checkValue(0, 110.00, "runScan"),
				checkValue(4, 30.00, "scanModule"),
			),
		},
		{
			name:    "top-3",
			entries: dataset,
			top:     3,
			checks: checkapplyFilters(
				checkLen(3),
				checkValue(0, 110.00, "runScan"),
				checkValue(2, 42.00, "applyFilters"),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := applyFilters(tt.entries.List, tt.top, tt.min)
			for _, c := range tt.checks {
				c(t, r)
			}
		})
	}
}

type ScanFn func(*testing.T, *score.EntryList, error)

var checkScan = func(fns ...ScanFn) []ScanFn { return fns }

func checkScanError(want string) ScanFn {
	return func(t *testing.T, _ *score.EntryList, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkScanError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkScanError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkScanError mismatch")
		}
	}
}

func TestScan(t *testing.T) {
	checkLen := func(count int) ScanFn {
		return func(t *testing.T, c *score.EntryList, e error) {
			t.Helper()
			if assert.NotNil(t, c) {
				assert.Equal(t, count, len(c.List))
			}
		}
	}

	checkValue := func(pos int, crap float64, name string) ScanFn {
		return func(t *testing.T, c *score.EntryList, e error) {
			t.Helper()
			if assert.NotNil(t, c) {
				assert.Equal(t, crap, c.List[pos].CRAP)
				assert.Equal(t, name, c.List[pos].FuncName)
			}
		}
	}

	checkSkipped := func(pos int, skipped bool) ScanFn {
		return func(t *testing.T, c *score.EntryList, e error) {
			t.Helper()
			if assert.NotNil(t, c) {
				assert.Equal(t, skipped, c.List[pos].Skipped)
			}
		}
	}

	tests := []struct {
		name    string
		options *Options
		checks  []ScanFn
	}{
		{
			name: "default scan",
			options: &Options{
				Path:    "../testdata",
				Exclude: []string{".*_test.go"},
			},
			checks: checkScan(
				checkScanError(""),
				checkLen(5),
				checkValue(0, 90.00, "veryComplex"),
				checkValue(4, 1.00, "simple"),
			),
		},
		{
			name: "missing skip marks all as skipped",
			options: &Options{
				Path:    "../testdata",
				Missing: "skip",
				Exclude: []string{".*_test.go"},
			},
			checks: checkScan(
				checkScanError(""),
				checkLen(5),
				checkSkipped(0, true),
				checkSkipped(3, true),
				checkSkipped(4, false),
			),
		},
		{
			name: "missing optimistic assumes 100% coverage",
			options: &Options{
				Path:    "../testdata",
				Missing: "optimistic",
			},
			checks: checkScan(
				checkScanError(""),
				checkLen(6),
			),
		},
		{
			name: "missing pessimistic default policy",
			options: &Options{
				Path:    "../testdata",
				Missing: "pessimistic",
			},
			checks: checkScan(
				checkScanError(""),
				checkLen(6),
			),
		},
		{
			name: "invalid missing policy returns error",
			options: &Options{
				Path:    "../testdata",
				Missing: "invalid",
			},
			checks: checkScan(
				checkScanError("unknown missing policy"),
			),
		},
		{
			name: "non-existent path returns error",
			options: &Options{
				Path: "/no/such/dir/that/does/not/exist",
			},
			checks: checkScan(
				checkScanError("coverage scan"),
			),
		},
		{
			name: "exclude function name reduces count",
			options: &Options{
				Path:    "../testdata",
				Exclude: []string{"veryComplex"},
			},
			checks: checkScan(
				checkScanError(""),
				checkLen(5),
				checkValue(0, 20.00, "withSwitch"),
			),
		},
		{
			name: "top 2 limits results",
			options: &Options{
				Path: "../testdata",
				Top:  2,
			},
			checks: checkScan(
				checkScanError(""),
				checkLen(2),
				checkValue(0, 90.00, "veryComplex"),
				checkValue(1, 20.00, "withSwitch"),
			),
		},
		{
			name: "min 50 filters low scores",
			options: &Options{
				Path: "../testdata",
				Min:  50,
			},
			checks: checkScan(
				checkScanError(""),
				checkLen(1),
				checkValue(0, 90.00, "veryComplex"),
			),
		},
		{
			name: "min higher than all scores returns empty",
			options: &Options{
				Path: "../testdata",
				Min:  100,
			},
			checks: checkScan(
				checkScanError(""),
				checkLen(0),
			),
		},
		{
			name: "top larger than result set is no-op",
			options: &Options{
				Path: "../testdata",
				Top:  100,
			},
			checks: checkScan(
				checkScanError(""),
				checkLen(6),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, err := Scan(tt.options)
			for _, c := range tt.checks {
				c(t, r, err)
			}
		})
	}
}
