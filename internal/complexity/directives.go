package complexity

import (
	"go/ast"
	"strings"
)

type directives struct {
	ignore bool
}

func parseDirectives(doc *ast.CommentGroup) directives {
	if doc == nil {
		return directives{}
	}
	for _, c := range doc.List {
		text := strings.TrimSpace(c.Text)
		if text == "//go-crap:ignore" || text == "//gocyclo:ignore" {
			return directives{ignore: true}
		}
	}
	return directives{}
}
