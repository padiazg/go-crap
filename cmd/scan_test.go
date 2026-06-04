package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			f.Value.Set(f.DefValue)
		}
	})
}

func checkOutputContains(want string) func(*testing.T, *bytes.Buffer, error) {
	return func(t *testing.T, buf *bytes.Buffer, err error) {
		t.Helper()
		if !assert.NoError(t, err, "runScan should not return an error") {
			return
		}
		assert.Containsf(t, buf.String(), want, "output should contain %q", want)
	}
}

func checkNoError() func(*testing.T, *bytes.Buffer, error) {
	return func(t *testing.T, buf *bytes.Buffer, err error) {
		t.Helper()
		assert.NoErrorf(t, err, "runScan should not return an error")
	}
}

func checkErrorContains(want string) func(*testing.T, *bytes.Buffer, error) {
	return func(t *testing.T, buf *bytes.Buffer, err error) {
		t.Helper()
		if assert.Error(t, err, "runScan should return an error") {
			assert.Containsf(t, err.Error(), want, "error should contain %q", want)
		}
	}
}

func Test_runScan(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		args    []string
		checks  []func(*testing.T, *bytes.Buffer, error)
	}{
		{
			name:   "default_path_table_format",
			format: "table",
			args:   []string{},
			checks: []func(*testing.T, *bytes.Buffer, error){
				checkNoError(),
				checkOutputContains("CRAP"),
				checkOutputContains("CC"),
				checkOutputContains("COVERAGE"),
				checkOutputContains("FUNCTION"),
			},
		},
		{
			name:   "custom_path_json_format",
			format: "json",
			args:   []string{"../internal/score"},
			checks: []func(*testing.T, *bytes.Buffer, error){
				checkNoError(),
				checkOutputContains("$schema"),
				checkOutputContains("entries"),
				checkOutputContains("score.go"),
			},
		},
		{
			name:   "custom_path_github_format",
			format: "github",
			args:   []string{"../internal/score"},
			checks: []func(*testing.T, *bytes.Buffer, error){
				checkNoError(),
				checkOutputContains("::warning"),
				checkOutputContains("score.go"),
			},
		},
		{
			name:   "nonexistent_path",
			format: "table",
			args:   []string{"./nonexistent/path"},
			checks: []func(*testing.T, *bytes.Buffer, error){
				checkErrorContains("coverage scan"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			oldFormat := flagFormat
			oldExclude := flagExclude
			defer func() {
				flagFormat = oldFormat
				flagExclude = oldExclude
			}()

			resetFlags(scanCmd)
			flagFormat = tt.format

			oldOut := scanCmd.OutOrStdout()
			scanCmd.SetOut(buf)
			defer scanCmd.SetOut(oldOut)

			err := runScan(scanCmd, tt.args)
			for _, c := range tt.checks {
				c(t, buf, err)
			}
		})
	}
}
