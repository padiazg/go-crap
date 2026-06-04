package utils

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchExclude(t *testing.T) {
	tests := []struct {
		name     string
		ignore   *regexp.Regexp
		filePath string
		want     bool
	}{
		{
			name:     "nil regex does not match",
			ignore:   nil,
			filePath: "main.go",
			want:     false,
		},
		{
			name:     "exact filename match",
			ignore:   regexp.MustCompile(`^main\.go$`),
			filePath: "main.go",
			want:     true,
		},
		{
			name:     "exact filename no match",
			ignore:   regexp.MustCompile(`^main\.go$`),
			filePath: "helpers.go",
			want:     false,
		},
		{
			name:     "suffix match at root",
			ignore:   regexp.MustCompile(`\.pb\.go$`),
			filePath: "main.pb.go",
			want:     true,
		},
		{
			name:     "suffix match nested",
			ignore:   regexp.MustCompile(`\.pb\.go$`),
			filePath: "pb/schema.pb.go",
			want:     true,
		},
		{
			name:     "suffix match no match",
			ignore:   regexp.MustCompile(`\.pb\.go$`),
			filePath: "main.go",
			want:     false,
		},
		{
			name:     "recursive test file match root",
			ignore:   regexp.MustCompile(`.*_test\.go$`),
			filePath: "main_test.go",
			want:     true,
		},
		{
			name:     "recursive test file match nested",
			ignore:   regexp.MustCompile(`.*_test\.go$`),
			filePath: "internal/pkg/main_test.go",
			want:     true,
		},
		{
			name:     "recursive test file match deep",
			ignore:   regexp.MustCompile(`.*_test\.go$`),
			filePath: "vendor/foo/bar/baz/test_test.go",
			want:     true,
		},
		{
			name:     "non-test file no match",
			ignore:   regexp.MustCompile(`.*_test\.go$`),
			filePath: "main.go",
			want:     false,
		},
		{
			name:     "mock pattern match",
			ignore:   regexp.MustCompile(`mock_`),
			filePath: "mock_client.go",
			want:     true,
		},
		{
			name:     "mock pattern no match",
			ignore:   regexp.MustCompile(`mock_`),
			filePath: "client.go",
			want:     false,
		},
		{
			name:     "directory prefix match",
			ignore:   regexp.MustCompile(`^pb/`),
			filePath: "pb/schema.pb.go",
			want:     true,
		},
		{
			name:     "directory prefix no match",
			ignore:   regexp.MustCompile(`^pb/`),
			filePath: "internal/pb/schema.pb.go",
			want:     false,
		},
		{
			name:     "any character in filename",
			ignore:   regexp.MustCompile(`main[0-9]+\.go`),
			filePath: "main123.go",
			want:     true,
		},
		{
			name:     "any character no match",
			ignore:   regexp.MustCompile(`main[0-9]+\.go`),
			filePath: "main.go",
			want:     false,
		},
		{
			name:     "function name match",
			ignore:   regexp.MustCompile(`TestHelper`),
			filePath: "TestHelper",
			want:     true,
		},
		{
			name:     "function name no match",
			ignore:   regexp.MustCompile(`TestHelper`),
			filePath: "doStuff",
			want:     false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := MatchExclude(tt.ignore, tt.filePath)
			assert.Equalf(t, tt.want, got, "MatchExclude(%v, %q)", tt.ignore, tt.filePath)
		})
	}
}

func TestMatchExclude_endToEnd(t *testing.T) {
	tests := []struct {
		name     string
		exclude  []string
		filePath string
		funcName string
		wantFile bool
		wantFunc bool
	}{
		{
			name:     "exclude test files at all depths",
			exclude:  []string{`.*_test\.go$`},
			filePath: "internal/pkg/main_test.go",
			funcName: "TestMain",
			wantFile: true,
			wantFunc: false,
		},
		{
			name:     "exclude protobuf files",
			exclude:  []string{`\.pb\.go$`},
			filePath: "vendor/foo/bar/schema.pb.go",
			funcName: "SomeFunc",
			wantFile: true,
			wantFunc: false,
		},
		{
			name:     "exclude mock files",
			exclude:  []string{`mock_`},
			filePath: "mock_repo.go",
			funcName: "mock_client",
			wantFile: true,
			wantFunc: true,
		},
		{
			name:     "exclude both test and protobuf",
			exclude:  []string{`.*_test\.go$`, `\.pb\.go$`},
			filePath: "main_test.pb.go",
			funcName: "SomeFunc",
			wantFile: true,
			wantFunc: false,
		},
		{
			name:     "exclude test files at root only",
			exclude:  []string{`^.*_test\.go$`},
			filePath: "main_test.go",
			funcName: "TestRoot",
			wantFile: true,
			wantFunc: false,
		},
		{
			name:     "exclude only root test files",
			exclude:  []string{`^[^/]*_test\.go$`},
			filePath: "vendor/pkg/main_test.go",
			funcName: "TestNested",
			wantFile: false,
			wantFunc: false,
		},
		{
			name:     "exclude only root test files matches root",
			exclude:  []string{`^[^/]*_test\.go$`},
			filePath: "main_test.go",
			funcName: "TestRoot",
			wantFile: true,
			wantFunc: false,
		},
		{
			name:     "no exclude list returns false",
			exclude:  nil,
			filePath: "anything.go",
			funcName: "Anything",
			wantFile: false,
			wantFunc: false,
		},
		{
			name:     "empty exclude list returns false",
			exclude:  []string{},
			filePath: "anything.go",
			funcName: "Anything",
			wantFile: false,
			wantFunc: false,
		},
		{
			name:     "exclude generated files",
			exclude:  []string{`generated\.go$`, `^//.*`},
			filePath: "api/generated.go",
			funcName: "GeneratedFunc",
			wantFile: true,
			wantFunc: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			re, err := BuildExcludeRegex(tt.exclude)
			if err != nil {
				assert.Failf(t, "BuildExcludeRegex", "%v", err)
				return
			}

			gotFile := MatchExclude(re, tt.filePath)
			assert.Equalf(t, tt.wantFile, gotFile, "MatchExclude file: %q", tt.filePath)

			gotFunc := MatchExclude(re, tt.funcName)
			assert.Equalf(t, tt.wantFunc, gotFunc, "MatchExclude func: %q", tt.funcName)
		})
	}
}

func TestBuildExcludeRegex(t *testing.T) {
	tests := []struct {
		name    string
		exclude []string
		wantNil bool
	}{
		{
			name:    "nil slice returns nil",
			exclude: nil,
			wantNil: true,
		},
		{
			name:    "empty slice returns nil",
			exclude: []string{},
			wantNil: true,
		},
		{
			name:    "single pattern compiles",
			exclude: []string{`main\.go`},
			wantNil: false,
		},
		{
			name:    "multiple patterns compile with alternation",
			exclude: []string{`mock_`, `pb/.*\.go`},
			wantNil: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildExcludeRegex(tt.exclude)
			if err != nil {
				assert.Failf(t, "BuildExcludeRegex", "%v", err)
				return
			}

			if tt.wantNil {
				assert.Nil(t, got, "BuildExcludeRegex(%v) should return nil", tt.exclude)
			} else {
				assert.NotNil(t, got, "BuildExcludeRegex(%v) should not return nil", tt.exclude)
			}
		})
	}
}

func TestBuildExcludeRegex_endToEnd(t *testing.T) {
	tests := []struct {
		name    string
		exclude []string
		path    string
		want    bool
	}{
		{
			name:    "test files excluded at all depths",
			exclude: []string{`.*_test\.go$`},
			path:    "vendor/pkg/helpers_test.go",
			want:    true,
		},
		{
			name:    "non-test files pass",
			exclude: []string{`.*_test\.go$`},
			path:    "vendor/pkg/helpers.go",
			want:    false,
		},
		{
			name:    "protobuf files excluded",
			exclude: []string{`\.pb\.go$`},
			path:    "pb/schema.pb.go",
			want:    true,
		},
		{
			name:    "mock files excluded",
			exclude: []string{`mock_`},
			path:    "mock_client.go",
			want:    true,
		},
		{
			name:    "mock files excluded from nested",
			exclude: []string{`mock_`},
			path:    "internal/pkg/mock_client.go",
			want:    true,
		},
		{
			name:    "no exclude passes",
			exclude: nil,
			path:    "anything.go",
			want:    false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			re, err := BuildExcludeRegex(tt.exclude)
			if err != nil {
				assert.Failf(t, "BuildExcludeRegex", "%v", err)
				return
			}

			got := MatchExclude(re, tt.path)
			assert.Equalf(t, tt.want, got, "path %q", tt.path)
		})
	}
}
