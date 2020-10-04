package minformat

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/token"
	"io"
)

// TODO: handle error (for invalid AST inputs)
// TODO: `t *T` => `t*T`
// TODO: `var x []int` => `var x[]int`
// TODO: `import "foo"` => `import"foo"`

type minifier struct {
	out  *bufio.Writer
	fset *token.FileSet
}

func (m *minifier) Fprint(w io.Writer, fset *token.FileSet, node interface{}) {
	m.fset = fset
	m.out = bufio.NewWriter(w)
	defer m.out.Flush()

	switch n := node.(type) {
	case ast.Node:
		m.printNode(n)
	default:
		m.panicUnhandled("Fprint", n)
	}
}

func (m *minifier) printNode(n ast.Node) {
	switch n := n.(type) {
	case *ast.File:
		fmt.Fprintf(m.out, "package %s;", n.Name.Name)
		for i, d := range n.Decls {
			m.printNode(d)
			if i != len(n.Decls)-1 {
				m.out.WriteByte(';')
			}
		}

	case ast.Decl:
		m.printDecl(n)
	case ast.Expr:
		m.printExpr(n)
	case ast.Stmt:
		m.printStmt(n)

	default:
		m.panicUnhandled("printNode", n)
	}
}

func (m *minifier) printDecl(n ast.Decl) {
	switch n := n.(type) {
	case *ast.FuncDecl:
		// TODO: methods
		if n.Recv != nil {
			m.out.WriteString("func(")
			m.printFieldList(n.Recv, ',')
			m.out.WriteByte(')')
		} else {
			m.out.WriteString("func ")
		}
		m.out.WriteString(n.Name.Name)
		m.printFuncType(n.Type)
		if n.Body != nil {
			m.printBlockStmt(n.Body)
		}

	case *ast.GenDecl:
		m.printGenDecl(n)

	default:
		m.panicUnhandled("printDecl", n)
	}
}

func (m *minifier) printExpr(n ast.Expr) {
	switch n := n.(type) {
	case *ast.Ident:
		m.out.WriteString(n.Name)

	case *ast.Ellipsis:
		m.out.WriteString("...")
		if n.Elt != nil {
			m.printExpr(n.Elt)
		}

	case *ast.ParenExpr:
		m.out.WriteByte('(')
		m.printExpr(n.X)
		m.out.WriteByte(')')

	case *ast.BasicLit:
		m.out.WriteString(n.Value)

	case *ast.IndexExpr:
		m.printExpr(n.X)
		m.out.WriteByte('[')
		m.printExpr(n.Index)
		m.out.WriteByte(']')

	case *ast.BinaryExpr:
		m.printBinaryExpr(n)

	case *ast.UnaryExpr:
		m.out.WriteString(n.Op.String())
		m.printExpr(n.X)

	case *ast.StarExpr:
		m.out.WriteByte('*')
		m.printExpr(n.X)

	case *ast.TypeAssertExpr:
		m.printExpr(n.X)
		m.out.WriteString(".(")
		if n.Type != nil {
			m.printExpr(n.Type)
		} else {
			m.out.WriteString("type")
		}
		m.out.WriteByte(')')

	case *ast.SelectorExpr:
		m.printExpr(n.X)
		m.out.WriteByte('.')
		m.out.WriteString(n.Sel.Name)

	case *ast.CallExpr:
		m.printExpr(n.Fun)
		m.out.WriteByte('(')
		for i, arg := range n.Args {
			m.printExpr(arg)
			if i != len(n.Args)-1 {
				m.out.WriteByte(',')
			}
		}
		m.out.WriteByte(')')

	case *ast.SliceExpr:
		m.printExpr(n.X)
		m.out.WriteByte('[')
		if n.Low != nil {
			m.printExpr(n.Low)
		}
		m.out.WriteRune(':')
		if n.High != nil {
			m.printExpr(n.High)
		}
		if n.Max != nil {
			m.out.WriteByte(':')
			m.printExpr(n.Max)
		}
		m.out.WriteByte(']')

	case *ast.CompositeLit:
		if n.Type != nil {
			m.printExpr(n.Type)
		}
		m.out.WriteByte('{')
		m.printExprList(n.Elts)
		m.out.WriteByte('}')

	case *ast.FuncLit:
		m.out.WriteString("func")
		m.printFuncType(n.Type)
		m.printBlockStmt(n.Body)

	case *ast.KeyValueExpr:
		m.printExpr(n.Key)
		m.out.WriteByte(':')
		m.printExpr(n.Value)

	case *ast.ChanType:
		switch {
		case n.Dir&ast.SEND != 0 && n.Dir&ast.RECV != 0:
			m.out.WriteString("chan ")
			m.printExpr(n.Value)
		case n.Dir&ast.SEND != 0:
			m.out.WriteString("chan<- ")
			m.printExpr(n.Value)
		case n.Dir&ast.RECV != 0:
			m.out.WriteString("<-chan ")
			m.printExpr(n.Value)
		}

	case *ast.ArrayType:
		m.out.WriteByte('[')
		if n.Len != nil {
			m.printExpr(n.Len)
		}
		m.out.WriteByte(']')
		m.printExpr(n.Elt)

	case *ast.MapType:
		m.out.WriteString("map[")
		m.printExpr(n.Key)
		m.out.WriteByte(']')
		m.printExpr(n.Value)

	case *ast.FuncType:
		if n.Func != token.NoPos {
			m.out.WriteString("func")
		}
		m.printFuncType(n)

	case *ast.StructType:
		m.printStructType(n)

	case *ast.InterfaceType:
		m.printInterfaceType(n)

	default:
		m.panicUnhandled("printExpr", n)
	}
}

func (m *minifier) printStmt(n ast.Stmt) {
	switch n := n.(type) {
	case *ast.EmptyStmt:
		if !n.Implicit {
			m.out.WriteByte(';')
		}

	case *ast.AssignStmt:
		for i, lhs := range n.Lhs {
			m.printExpr(lhs)
			if i != len(n.Lhs)-1 {
				m.out.WriteByte(',')
			}
		}
		m.out.WriteString(n.Tok.String())
		for i, rhs := range n.Rhs {
			m.printExpr(rhs)
			if i != len(n.Rhs)-1 {
				m.out.WriteByte(',')
			}
		}

	case *ast.IncDecStmt:
		m.printExpr(n.X)
		m.out.WriteString(n.Tok.String())

	case *ast.BranchStmt:
		m.out.WriteString(n.Tok.String())
		if n.Label != nil {
			m.out.WriteByte(' ')
			m.out.WriteString(n.Label.Name)
		}

	case *ast.RangeStmt:
		switch {
		case n.Key == nil && n.Value == nil:
			m.out.WriteString("for range ")
			m.printExpr(n.X)
			m.printBlockStmt(n.Body)
		case n.Key != nil && n.Value == nil:
			m.out.WriteString("for ")
			m.printExpr(n.Key)
			m.out.WriteString(n.Tok.String())
			m.out.WriteString("range ")
			m.printExpr(n.X)
			m.printBlockStmt(n.Body)
		default:
			m.out.WriteString("for ")
			m.printExpr(n.Key)
			m.out.WriteByte(',')
			m.printExpr(n.Value)
			m.out.WriteString(n.Tok.String())
			m.out.WriteString("range ")
			m.printExpr(n.X)
			m.printBlockStmt(n.Body)
		}

	case *ast.ForStmt:
		switch {
		case n.Init == nil && n.Cond == nil && n.Post == nil:
			m.out.WriteString("for")
			m.printBlockStmt(n.Body)
		case n.Init == nil && n.Cond != nil && n.Post == nil:
			m.out.WriteString("for ")
			m.printExpr(n.Cond)
			m.printBlockStmt(n.Body)
		default:
			m.out.WriteString("for ")
			if n.Init != nil {
				m.printStmt(n.Init)
			}
			m.out.WriteByte(';')
			if n.Cond != nil {
				m.printExpr(n.Cond)
			}
			m.out.WriteByte(';')
			if n.Post != nil {
				m.printStmt(n.Post)
			}
			m.printBlockStmt(n.Body)
		}

	case *ast.ReturnStmt:
		m.out.WriteString("return ")
		for i, x := range n.Results {
			m.printExpr(x)
			if i != len(n.Results)-1 {
				m.out.WriteByte(',')
			}
		}

	case *ast.ExprStmt:
		m.printExpr(n.X)

	case *ast.DeclStmt:
		m.printDecl(n.Decl)

	case *ast.LabeledStmt:
		m.out.WriteString(n.Label.Name)
		m.out.WriteByte(':')
		m.printStmt(n.Stmt)

	case *ast.SwitchStmt:
		if n.Init == nil && n.Tag == nil {
			m.out.WriteString("switch")
			m.printBlockStmt(n.Body)
			return
		}
		m.out.WriteString("switch ")
		if n.Init != nil {
			m.printStmt(n.Init)
			m.out.WriteByte(';')
		}
		if n.Tag != nil {
			m.printExpr(n.Tag)
		}
		m.printBlockStmt(n.Body)

	case *ast.TypeSwitchStmt:
		m.out.WriteString("switch ")
		if n.Init != nil {
			m.printStmt(n.Init)
			m.out.WriteByte(';')
		}
		m.printStmt(n.Assign)
		m.printBlockStmt(n.Body)

	case *ast.SelectStmt:
		m.out.WriteString("select")
		m.printBlockStmt(n.Body)

	case *ast.SendStmt:
		m.printExpr(n.Chan)
		m.out.WriteString("<-")
		m.printExpr(n.Value)

	case *ast.CommClause:
		if n.Comm == nil {
			m.out.WriteString("default:")
		} else {
			m.out.WriteString("case ")
			m.printStmt(n.Comm)
			m.out.WriteByte(':')
		}
		m.printStmtList(n.Body)

	case *ast.CaseClause:
		if n.List == nil {
			m.out.WriteString("default:")
		} else {
			m.out.WriteString("case ")
			for i, x := range n.List {
				m.printExpr(x)
				if i != len(n.List)-1 {
					m.out.WriteByte(',')
				}
			}
			m.out.WriteByte(':')
		}
		m.printStmtList(n.Body)

	case *ast.IfStmt:
		m.out.WriteString("if ")
		if n.Init != nil {
			m.printStmt(n.Init)
			m.out.WriteByte(';')
		}
		m.printExpr(n.Cond)
		m.printBlockStmt(n.Body)

	case *ast.BlockStmt:
		m.printBlockStmt(n)

	case *ast.DeferStmt:
		m.out.WriteString("defer ")
		m.printExpr(n.Call)

	case *ast.GoStmt:
		m.out.WriteString("go ")
		m.printExpr(n.Call)

	default:
		m.panicUnhandled("printStmt", n)
	}
}

func (m *minifier) printBlockStmt(n *ast.BlockStmt) {
	m.out.WriteByte('{')
	m.printStmtList(n.List)
	m.out.WriteByte('}')
}

func (m *minifier) printStmtList(list []ast.Stmt) {
	for i, stmt := range list {
		m.printStmt(stmt)
		if i != len(list)-1 {
			m.out.WriteByte(';')
		}
	}
}

func (m *minifier) printExprList(list []ast.Expr) {
	for i, expr := range list {
		m.printExpr(expr)
		if i != len(list)-1 {
			m.out.WriteByte(',')
		}
	}
}

func (m *minifier) printGenDecl(n *ast.GenDecl) {
	m.out.WriteString(n.Tok.String())
	if n.Lparen != token.NoPos {
		m.out.WriteByte('(')
	} else {
		m.out.WriteByte(' ')
	}
	for i, spec := range n.Specs {
		switch spec := spec.(type) {
		case *ast.ImportSpec:
			if spec.Name != nil {
				m.out.WriteString(spec.Name.Name)
				// Note: space is not needed.
			}
			m.printExpr(spec.Path)
		case *ast.ValueSpec:
			for i, ident := range spec.Names {
				m.out.WriteString(ident.Name)
				if i != len(spec.Names)-1 {
					m.out.WriteByte(',')
				}
			}
			if spec.Type != nil {
				m.out.WriteByte(' ')
				m.printExpr(spec.Type)
			}
			if spec.Values != nil {
				m.out.WriteByte('=')
				for i, x := range spec.Values {
					m.printExpr(x)
					if i != len(spec.Values)-1 {
						m.out.WriteByte(',')
					}
				}
			}
		case *ast.TypeSpec:
			m.out.WriteString(spec.Name.Name)
			if spec.Assign != token.NoPos {
				m.out.WriteByte('=')
			} else {
				m.out.WriteByte(' ')
			}
			m.printExpr(spec.Type)

		default:
			panic("unreachable")
		}
		if i != len(n.Specs)-1 {
			m.out.WriteByte(';')
		}
	}
	if n.Rparen != token.NoPos {
		m.out.WriteByte(')')
	}
}

func (m *minifier) printBinaryExpr(n *ast.BinaryExpr) {
	// Handle `x < -y` and `x - -y`.
	if n.Op == token.LSS || n.Op == token.SUB {
		y := leftmostExpr(n.Y)
		if y, ok := y.(*ast.UnaryExpr); ok && y.Op == token.SUB {
			m.printExpr(n.X)
			m.out.WriteString(n.Op.String())
			m.out.WriteByte(' ')
			m.printExpr(n.Y)
			return
		}
	}

	m.printExpr(n.X)
	m.out.WriteString(n.Op.String())
	m.printExpr(n.Y)
}

func (m *minifier) printFuncType(n *ast.FuncType) {
	m.out.WriteString("(")
	m.printFieldList(n.Params, ',')
	m.out.WriteString(")")
	if n.Results != nil {
		needParens := !(len(n.Results.List) == 1 && len(n.Results.List[0].Names) == 0)
		if needParens {
			m.out.WriteString("(")
		}
		m.printFieldList(n.Results, ',')
		if needParens {
			m.out.WriteString(")")
		}
	}
}

func (m *minifier) printStructType(n *ast.StructType) {
	m.out.WriteString("struct{")
	m.printFieldList(n.Fields, ';')
	m.out.WriteByte('}')
}

func (m *minifier) printInterfaceType(n *ast.InterfaceType) {
	m.out.WriteString("interface{")

	for j, field := range n.Methods.List {
		if len(field.Names) == 1 {
			m.out.WriteString(field.Names[0].Name)
		}
		m.printExpr(field.Type)
		if j != len(n.Methods.List)-1 {
			m.out.WriteByte(';')
		}
	}

	m.out.WriteByte('}')
}

func (m *minifier) printFieldList(n *ast.FieldList, sep byte) {
	for j, field := range n.List {
		for j, ident := range field.Names {
			m.out.WriteString(ident.Name)
			if j != len(field.Names)-1 {
				m.out.WriteString(",")
			}
		}
		if len(field.Names) != 0 {
			m.out.WriteByte(' ')
		}
		m.printExpr(field.Type)
		if j != len(n.List)-1 {
			m.out.WriteByte(sep)
		}
	}
}

func (m *minifier) panicUnhandled(fn string, n interface{}) {
	if n, ok := n.(ast.Node); ok {
		pos := m.fset.Position(n.Pos())
		panic(fmt.Sprintf("%s:%d: %s: unhandled %T", pos.Filename, pos.Line, fn, n))
	}
	panic(fmt.Sprintf("<?>: %s: unhandled %T", fn, n))
}
