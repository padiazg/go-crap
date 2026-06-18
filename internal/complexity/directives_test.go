package complexity

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go/ast"
)

func Test_parseDirectives(t *testing.T) {
	tests := []struct {
		name string
		doc  *ast.CommentGroup
		want bool
	}{
		{name: "nil doc", doc: nil, want: false},
		{name: "empty comment group", doc: &ast.CommentGroup{}, want: false},
		{name: "empty comment list", doc: &ast.CommentGroup{List: []*ast.Comment{}}, want: false},
		{name: "//go-crap:ignore", doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "//go-crap:ignore"}}}, want: true},
		{name: "//gocyclo:ignore", doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "//gocyclo:ignore"}}}, want: true},
		{name: "//go-crap:ignore with leading space returns false", doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "//  go-crap:ignore"}}}, want: false},
		{name: "//gocyclo:ignore with leading space returns false", doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "//  gocyclo:ignore"}}}, want: false},
		{name: "//go-crap:ignore with trailing space", doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "//go-crap:ignore  "}}}, want: true},
		{name: "other directive", doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "// some comment"}}}, want: false},
		{name: "mismatched case", doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "//GO-CRAP:IGNORE"}}}, want: false},
		{name: "ignore among other comments", doc: &ast.CommentGroup{List: []*ast.Comment{
			{Text: "// first comment"},
			{Text: "//go-crap:ignore"},
			{Text: "// last comment"},
		}}, want: true},
		{name: "gocyclo among other comments", doc: &ast.CommentGroup{List: []*ast.Comment{
			{Text: "// doc line 1"},
			{Text: "//gocyclo:ignore"},
			{Text: "// doc line 3"},
		}}, want: true},
		{name: "similar but not matching", doc: &ast.CommentGroup{List: []*ast.Comment{
			{Text: "//go-crap:ignored"},
			{Text: "//gocyclo:ignorex"},
		}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := parseDirectives(tt.doc)
			assert.Equal(t, tt.want, r)
		})
	}
}
