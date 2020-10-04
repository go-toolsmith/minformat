// Package minformat implements Go source code minification routines.
package minformat

import (
	"bytes"
	"go/parser"
	"go/token"
	"io"
)

// Node formats node by removing as much whitespace as possible and writes the result to w.
//
// Result contains no comments.
//
// The node type is defined as interface{} for compatibility with go/format.Node function.
// Only ast.Node types are supported right now.
//
// The function may return early (before the entire result is written) and return a formatting error,
// for instance due to an incorrect AST.
func Node(w io.Writer, fset *token.FileSet, node interface{}) error {
	var m minifier
	m.Fprint(w, fset, node)
	return nil
}

// Source formats src by removing as mych whitespace as possible and returns the result.
//
// src is expected to be a syntactically correct Go source file.
func Source(src []byte) ([]byte, error) {
	const parserMode = 0
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "source-input", src, parserMode)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = Node(&buf, fset, f)
	return buf.Bytes(), err
}
