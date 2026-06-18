package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

type runVersionCheckFn func(*testing.T)

var checkrunVersion = func(fns ...runVersionCheckFn) []runVersionCheckFn { return fns }

func Test_runVersion(t *testing.T) {
	tests := []struct {
		name   string
		cmd    *cobra.Command
		args   []string
		checks []runVersionCheckFn
	}{
		{
			name:   "TODO: success case",
			checks: checkrunVersion(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			runVersion(tt.cmd, tt.args)
			for _, c := range tt.checks {
				c(t)
			}
		})
	}
}
