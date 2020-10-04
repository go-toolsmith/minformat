package minformat

import (
	"go/ast"
)

func leftmostExpr(n ast.Expr) ast.Expr {
	switch n := n.(type) {
	case *ast.BinaryExpr:
		return leftmostExpr(n.X)
	default:
		return n
	}
}
