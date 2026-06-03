package complexity

import (
	"go/ast"
	"go/token"
)

func Complexity(fn ast.Node) int {
	v := complexityVisitor{complexity: 1}
	ast.Walk(&v, fn)
	return v.complexity
}

type complexityVisitor struct {
	complexity int
}

func (v *complexityVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
		v.complexity++
	case *ast.CaseClause:
		if n.List != nil {
			v.complexity++
		}
	case *ast.CommClause:
		if n.Comm != nil {
			v.complexity++
		}
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			v.complexity++
		}
	}
	return v
}
