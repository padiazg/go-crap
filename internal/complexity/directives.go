package complexity

import (
	"go/ast"
	"strings"
)

func parseDirectives(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}

	for _, c := range doc.List {
		text := strings.TrimSpace(c.Text)
		if (text == "//go-crap:ignore") || (text == "//gocyclo:ignore") {
			return true
		}
	}

	return false
}
