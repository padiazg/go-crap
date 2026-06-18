package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
	"github.com/stretchr/testify/assert"
)

func TestJSONFormatter_DetailedMutationDetails(t *testing.T) {
	entries := scan.Entries{List: []score.CRAPEntry{
		{
			File:              "/home/user/project/main.go",
			Package:           "myapp",
			FuncName:          "BadFunction",
			Receiver:          "",
			Line:              42,
			Complexity:        10,
			Coverage:          90.0,
			CRAP:              12.0,
			EffectiveCRAP:     120.0,
			CoverageUntrusted: true,
			MutationScore:     0.5,
			MutationDetails: []score.MutationDetail{
				{MutantType: "CONDITIONALS_BOUNDARY", MutatorName: "CB", File: "main.go", Line: 45, Status: "LIVED", OriginalText: "a < b", ReplacementText: "a >= b"},
				{MutantType: "ARITHMETIC", MutatorName: "SUB", File: "main.go", Line: 48, Status: "LIVED", OriginalText: "a + b", ReplacementText: "a - b"},
			},
		},
		{
			File:              "/home/user/project/other.go",
			Package:           "myapp",
			FuncName:          "GoodFunction",
			Receiver:          "",
			Line:              10,
			Complexity:        3,
			Coverage:          100.0,
			CRAP:              3.0,
			EffectiveCRAP:     3.0,
			CoverageUntrusted: false,
			MutationScore:     1.0,
			MutationDetails:   []score.MutationDetail{},
		},
	}}

	var gotReport Report
	buf := &bytes.Buffer{}
	opts := FormatOptions{
		Writer:   buf,
		BaseDir:  "/home/user/project",
		Detailed: true,
	}

	s := &JSONFormatter{jsonMarshalIndent: func(v any, prefix, indent string) ([]byte, error) {
		data, err := json.MarshalIndent(v, prefix, indent)
		if err == nil {
			_ = json.Unmarshal(data, &gotReport)
		}
		return data, err
	}}

	err := s.Format(&entries, opts)
	assert.NoError(t, err)

	assert.Equal(t, "myapp", gotReport.Entries[0].Package)
	assert.Equal(t, 2, len(gotReport.Entries))

	badEntry := gotReport.Entries[0]
	assert.Equal(t, "BadFunction", badEntry.Function)
	assert.Equal(t, 2, len(badEntry.MutationDetails))
	assert.Equal(t, "CONDITIONALS_BOUNDARY", badEntry.MutationDetails[0].Type)
	assert.Equal(t, "CB", badEntry.MutationDetails[0].MutatorName)
	assert.Equal(t, 45, badEntry.MutationDetails[0].Line)
	assert.Equal(t, "LIVED", badEntry.MutationDetails[0].Status)
	assert.Equal(t, "a < b", badEntry.MutationDetails[0].OriginalText)
	assert.Equal(t, "a >= b", badEntry.MutationDetails[0].ReplacementText)

	assert.Equal(t, "ARITHMETIC", badEntry.MutationDetails[1].Type)
	assert.Equal(t, 48, badEntry.MutationDetails[1].Line)

	goodEntry := gotReport.Entries[1]
	assert.Equal(t, "GoodFunction", goodEntry.Function)
	assert.Nil(t, goodEntry.MutationDetails)
}

type checkJSONFormatterFormatFn func(*testing.T, error)

var checkJSONFormatterFormat = func(fns ...checkJSONFormatterFormatFn) []checkJSONFormatterFormatFn { return fns }

type checkJSONFormatterFormatReportFn func(*testing.T, Report)

var checkJSONFormatterFormatReport = func(fns ...checkJSONFormatterFormatReportFn) []checkJSONFormatterFormatReportFn {
	return fns
}

func checkReportSchema(want string) checkJSONFormatterFormatReportFn {
	return func(t *testing.T, r Report) {
		t.Helper()
		assert.Equalf(t, want, r.Schema, "Report.Schema mismatch")
	}
}
func checkReportVersion(want string) checkJSONFormatterFormatReportFn {
	return func(t *testing.T, r Report) {
		t.Helper()
		assert.Equalf(t, want, r.Version, "Report.Version mismatch")
	}
}
func checkReportEntriesLen(want int) checkJSONFormatterFormatReportFn {
	return func(t *testing.T, r Report) {
		t.Helper()
		assert.Equalf(t, want, len(r.Entries), "len(Entries) mismatch")
	}
}
func checkReportEntries(i int, fns ...func(*testing.T, JSONEntry)) checkJSONFormatterFormatReportFn {
	return func(t *testing.T, r Report) {
		t.Helper()
		t.Helper()
		assert.GreaterOrEqualf(t, len(r.Entries), i+1, "Entries has enough items at index %d", i)
		entry := r.Entries[i]
		for _, fn := range fns {
			fn(t, entry)
		}
	}
}

func checkEntryFile(want string) func(*testing.T, JSONEntry) {
	return func(t *testing.T, e JSONEntry) {
		t.Helper()
		assert.Equalf(t, want, e.File, "entry.File mismatch")
	}
}
func checkEntryPackage(want string) func(*testing.T, JSONEntry) {
	return func(t *testing.T, e JSONEntry) {
		t.Helper()
		assert.Equalf(t, want, e.Package, "entry.Package mismatch")
	}
}
func checkEntryFunction(want string) func(*testing.T, JSONEntry) {
	return func(t *testing.T, e JSONEntry) {
		t.Helper()
		assert.Equalf(t, want, e.Function, "entry.Function mismatch")
	}
}
func checkEntryLine(want int) func(*testing.T, JSONEntry) {
	return func(t *testing.T, e JSONEntry) {
		t.Helper()
		assert.Equalf(t, want, e.Line, "entry.Line mismatch")
	}
}
func checkEntryCyclomatic(want int) func(*testing.T, JSONEntry) {
	return func(t *testing.T, e JSONEntry) {
		t.Helper()
		assert.Equalf(t, want, e.Cyclomatic, "entry.Cyclomatic mismatch")
	}
}
func checkEntryCRAP(want float64) func(*testing.T, JSONEntry) {
	return func(t *testing.T, e JSONEntry) {
		t.Helper()
		assert.InDeltaf(t, want, e.CRAP, 0.01, "entry.CRAP mismatch")
	}
}
func checkEntryCoverage(want float64) func(*testing.T, JSONEntry) {
	return func(t *testing.T, e JSONEntry) {
		t.Helper()
		assert.NotNil(t, e.Coverage, "Coverage should not be nil")
		assert.InDeltaf(t, want, *e.Coverage, 0.01, "entry.Coverage mismatch")
	}
}
func checkEntryReceiverNilOrEmpty() func(*testing.T, JSONEntry) {
	return func(t *testing.T, e JSONEntry) {
		t.Helper()
		// omitempty should have omitted it, but struct value will be ""
		assert.Emptyf(t, e.Receiver, "entry.Receiver should be empty")
	}
}

func checkFormatError(want string) checkJSONFormatterFormatFn {
	return func(t *testing.T, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkFormatError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkFormatError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkFormatError mismatch")
		}
	}
}
func TestJSONFormatter_Format(t *testing.T) {
	tests := []struct {
		name        string
		entries     scan.Entries
		opts        FormatOptions
		checks      []checkJSONFormatterFormatFn
		reportCheck []checkJSONFormatterFormatReportFn
		before      func(*JSONFormatter)
	}{
		{
			name: "success_empty_entries",
			reportCheck: checkJSONFormatterFormatReport(
				checkReportSchema("https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json"),
				checkReportVersion("1.0.0"),
				checkReportEntriesLen(0),
			),
		},
		{
			name: "success_single_entry",
			entries: scan.Entries{List: []score.CRAPEntry{
				{
					File:       "/home/user/project/main.go",
					Package:    "myapp",
					FuncName:   "HelloWorld",
					Receiver:   "",
					Line:       42,
					Complexity: 5,
					Coverage:   80.0,
					CRAP:       23.0,
				},
			}},
			reportCheck: checkJSONFormatterFormatReport(
				checkReportSchema("https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json"),
				checkReportVersion("1.0.0"),
				checkReportEntriesLen(1),
				checkReportEntries(0,
					checkEntryFile("/home/user/project/main.go"),
					checkEntryPackage("myapp"),
					checkEntryFunction("HelloWorld"),
					checkEntryLine(42),
					checkEntryCyclomatic(5),
					checkEntryCRAP(23.0),
					checkEntryCoverage(80.0),
				),
			),
		},
		{
			name: "success_receiver_omitempty",
			entries: scan.Entries{List: []score.CRAPEntry{
				{
					File:       "/home/user/project/main.go",
					Package:    "myapp",
					FuncName:   "HelloWorld",
					Receiver:   "",
					Line:       42,
					Complexity: 5,
					Coverage:   80.0,
					CRAP:       23.0,
				},
			}},
			reportCheck: checkJSONFormatterFormatReport(
				checkReportEntriesLen(1),
				checkReportEntries(0,
					checkEntryReceiverNilOrEmpty(),
				),
			),
		},
		{
			name: "success_coverage_zero_included",
			entries: scan.Entries{List: []score.CRAPEntry{
				{
					File:       "/home/user/project/main.go",
					Package:    "myapp",
					FuncName:   "NewConnection",
					Receiver:   "",
					Line:       10,
					Complexity: 3,
					Coverage:   0.0,
					CRAP:       12.0,
				},
			}},
			reportCheck: checkJSONFormatterFormatReport(
				checkReportEntriesLen(1),
				checkReportEntries(0,
					checkEntryFile("/home/user/project/main.go"),
					checkEntryCoverage(0.0),
				),
			),
		},
		{
			name: "success_base_dir_rewrites_path",
			entries: scan.Entries{List: []score.CRAPEntry{
				{
					File:       "/tmp/project/main.go",
					Package:    "myapp",
					FuncName:   "Process",
					Receiver:   "",
					Line:       5,
					Complexity: 2,
					Coverage:   90.0,
					CRAP:       7.2,
				},
			}},
			opts: FormatOptions{BaseDir: "/tmp/project"},
			reportCheck: checkJSONFormatterFormatReport(
				checkReportEntriesLen(1),
				checkReportEntries(0,
					checkEntryFile("main.go"),
					checkEntryPackage("myapp"),
				),
			),
		},
		{
			name: "error_marshal",
			before: func(j *JSONFormatter) {
				j.jsonMarshalIndent = func(v any, prefix, indent string) ([]byte, error) { return nil, fmt.Errorf("json-marshalindent-error") }
			},
			checks: checkJSONFormatterFormat(
				checkFormatError("json-marshalindent-error"),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := &JSONFormatter{}
			var gotReport Report
			buf := &bytes.Buffer{}
			opts := tt.opts
			if opts.Writer == nil {
				opts.Writer = buf
			}
			defaultMarshal := func(v any, prefix, indent string) ([]byte, error) {
				return json.MarshalIndent(v, prefix, indent)
			}
			captured := func(v any, prefix, indent string) ([]byte, error) {
				data, err := json.MarshalIndent(v, prefix, indent)
				if err == nil && tt.reportCheck != nil {
					_ = json.Unmarshal(data, &gotReport)
				}
				return data, err
			}
			if tt.before != nil {
				tt.before(s)
				orig := s.jsonMarshalIndent
				if orig == nil {
					orig = defaultMarshal
				}
				s.jsonMarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
					data, err := orig(v, prefix, indent)
					if err == nil && tt.reportCheck != nil {
						_ = json.Unmarshal(data, &gotReport)
					}
					return data, err
				}
			} else {
				s.jsonMarshalIndent = captured
			}
			err := s.Format(&tt.entries, opts)
			for _, c := range tt.checks {
				c(t, err)
			}
			for _, c := range tt.reportCheck {
				c(t, gotReport)
			}
		})
	}
}

func TestJSONFormatter_nil_entries(t *testing.T) {
	formatter := NewJSONFormatter()
	var buf strings.Builder
	err := formatter.Format(nil, FormatOptions{Writer: &buf})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestJSONFormatter_coverage_zero_in_output(t *testing.T) {
	formatter := NewJSONFormatter()
	var buf strings.Builder
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 10.0, Coverage: 0.0, CoverageUntrusted: false, FuncName: "zeroCover"},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "coverage")
	assert.Contains(t, output, "0")
}

func TestJSONFormatter_coverage_value_precision(t *testing.T) {
	formatter := NewJSONFormatter()
	var buf strings.Builder
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 10.0, Coverage: 50.5, CoverageUntrusted: false, FuncName: "hasCov"},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "coverage")
	assert.Contains(t, output, "50.5")
}

func TestJSONFormatter_entry_with_receiver(t *testing.T) {
	formatter := NewJSONFormatter()
	var buf strings.Builder
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 20.0, Coverage: 50.0, CoverageUntrusted: false, FuncName: "Method", Receiver: "MyType"},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, `"receiver"`)
	assert.Contains(t, output, "MyType")
}

func TestJSONFormatter_detailed_disabled_omits_mutations(t *testing.T) {
	formatter := NewJSONFormatter()
	var buf strings.Builder
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 50.0, Coverage: 50.0, CoverageUntrusted: true, FuncName: "untrusted",
			MutationScore: 0.5, MutationDetails: []score.MutationDetail{
				{MutantType: "CONDITIONALS_BOUNDARY", Line: 10, Status: "lived"},
			}},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Detailed: false})
	assert.NoError(t, err)
	output := buf.String()
	assert.NotContains(t, output, "mutation_details")
}

func TestJSONFormatter_convertToJSONEntry_emptyMutationDetails_Nil(t *testing.T) {
	f := &JSONFormatter{}
	entry := f.convertToJSONEntry(score.CRAPEntry{
		File:              "/main.go",
		Package:           "mypkg",
		FuncName:          "Foo",
		Complexity:        1,
		Coverage:          100.0,
		CRAP:              1.0,
		EffectiveCRAP:     1.0,
		MutationScore:     1.0,
		CoverageUntrusted: false,
		MutationDetails:   []score.MutationDetail{},
	}, FormatOptions{Detailed: true})
	assert.Nil(t, entry.MutationDetails, "empty MutationDetails should not be set when len==0")
}
