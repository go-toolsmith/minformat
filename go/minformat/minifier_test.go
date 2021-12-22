package minformat

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-toolsmith/strparse"
	"github.com/google/go-cmp/cmp"
)

func TestMinifyDecl(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{`func f() {}`, `func f(){}`},
		{`func (*T) m() {}`, `func(*T)m(){}`},
		{`func (t *T) m(int, int) (int , int) {}`, `func(t *T)m(int,int)(int,int){}`},

		{`type x = int`, `type x=int`},
		{`type (a [2] int; b [ ]int)`, `type(a [2]int;b []int)`},

		{`type _ struct { x int "struct tag" }`, `type _ struct{x int"struct tag"}`},

		{`const x, y = 1, 2`, `const x,y=1,2`},
		{`const (x = 1; y = 2)`, `const(x=1;y=2)`},
		{`const x int = 1`, `const x int=1`},

		{`var x, y = 1, 2`, `var x,y=1,2`},
		{`var (x = 1; y = 2)`, `var(x=1;y=2)`},
		{`var x, y [ ]int = nil, nil`, `var x,y []int=nil,nil`},
		{`var x, y [ ]int`, `var x,y []int`},

		{`import "foo"`, `import "foo"`},
		{`import ("foo")`, `import("foo")`},
		{`import ("foo"; a "b")`, `import("foo";a"b")`},
	}

	var m minifier
	for _, test := range tests {
		stmt := strparse.Decl(test.src)
		var buf bytes.Buffer
		m.Fprint(&buf, token.NewFileSet(), stmt)
		have := buf.Bytes()
		if !bytes.Equal(have, []byte(test.want)) {
			t.Errorf("minify %s:\nhave: %q\nwant: %q", test.src, have, test.want)
		}
	}
}

func TestMinifyStmt(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{`{ ; }`, `{;}`},

		{`x ++`, `x++`},
		{`x [0 ] --`, `x[0]--`},

		{`x = f()`, `x=f()`},
		{`x, y = f()`, `x,y=f()`},
		{`x, y = a, b`, `x,y=a,b`},

		{`{ }`, `{}`},
		{`{ 1; }`, `{1}`},
		{`{ 1; 2 }`, `{1;2}`},

		{`defer f()`, `defer f()`},
		{`defer f(1, 2)`, `defer f(1,2)`},
		{`defer func () {}()`, `defer func(){}()`},

		{`go f()`, `go f()`},
		{`go f(1, 2)`, `go f(1,2)`},

		{`label: f()`, `label:f()`},
		{`break`, `break`},
		{`break foo`, `break foo`},
		{`goto  label`, `goto label`},

		{`if cond { return nil }`, `if cond{return nil}`},
		{`if x := f(); !x { return nil }`, `if x:=f();!x{return nil}`},
		{`if cond { f(); } else { g(); }`, `if cond{f()}else{g()}`},
		{`if cond { f(); } else if cond { g(); }`, `if cond{f()}else if cond{g()}`},

		{`switch {default: return 1}`, `switch{default:return 1}`},
		{`switch tag {case 1, 2: return 0}`, `switch tag{case 1,2:return 0}`},

		{`switch x := x.(type) { }`, `switch x:=x.(type){}`},
		{`switch x := x.(type) {case int, float32:}`, `switch x:=x.(type){case int,float32:}`},
		{`switch x:=10; {default: return x}`, `switch x:=10;{default:return x}`},
		{`switch x. ( type ) {}`, `switch x.(type){}`},
		{`switch x := v; x := x.(type) { }`, `switch x:=v;x:=x.(type){}`},

		{`for {}`, `for{}`},
		{`for cond {}`, `for cond{}`},
		{`for ;; {}`, `for{}`},
		{`for i := 0;; {}`, `for i:=0;;{}`},
		{`for ; i < len(xs); {}`, `for i<len(xs){}`},
		{`for ;;i++ {}`, `for ;;i++{}`},
		{`for i := 0; ; i++ {}`, `for i:=0;;i++{}`},
		{`for i := 0; i < len(xs); i++ {}`, `for i:=0;i<len(xs);i++{}`},

		{`for range xs {}`, `for range xs{}`},
		{`for i := range xs {}`, `for i:=range xs{}`},
		{`for i, x = range xs {}`, `for i,x=range xs{}`},

		{`ch <- v`, `ch<-v`},
		{`ch[ 0 ] <- v.x`, `ch[0]<-v.x`},

		{`select {}`, `select{}`},
		{`select {default:}`, `select{default:}`},
		{`select {case <-ch: return 10}`, `select{case <-ch:return 10}`},
		{`select {case x := <-ch: return x}`, `select{case x:=<-ch:return x}`},
		{`select {case <-ch: return 10; default: return 0}`, `select{case <-ch:return 10;default:return 0}`},
	}

	var m minifier
	for _, test := range tests {
		stmt := strparse.Stmt(test.src)
		var buf bytes.Buffer
		m.Fprint(&buf, token.NewFileSet(), stmt)
		have := buf.Bytes()
		if !bytes.Equal(have, []byte(test.want)) {
			t.Errorf("minify %s:\nhave: %q\nwant: %q", test.src, have, test.want)
		}
	}
}

func TestMinifyExpr(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{`+ 1`, `+1`},
		{`* x`, `*x`},
		{`1 + 2`, `1+2`},
		{`1-2 - 3`, `1-2-3`},
		{` ( x ) `, `(x)`},
		{`"x" > "y"`, `"x">"y"`},
		{`x . y[0] . z`, `x.y[0].z`},
		{`<- x`, `<-x`},

		{`x.( int )`, `x.(int)`},
		{`x.( type )`, `x.(type)`},
		{`x . (a) . (b)`, `x.(a).(b)`},

		{`s[ : ]`, `s[:]`},
		{`s [ 1: ]`, `s[1:]`},
		{`s [ :len(s)-1 ]`, `s[:len(s)-1]`},
		{`s[ a:b ]`, `s[a:b]`},
		{`s[ a:b: c ]`, `s[a:b:c]`},
		{` s[ :b:c] `, `s[:b:c]`},

		{`[] int`, `[]int`},
		{`[ 2 ] int`, `[2]int`},

		{`x &^ y`, `x&^y`},
		{`x & ^y`, `x& ^y`},
		{`x & (^y)`, `x&(^y)`},

		{`func () `, `func()`},
		{`func ( int ) (int, int)`, `func(int)(int,int)`},
		{`func ( int, int ) (int)`, `func(int,int)int`},
		{`func ( int, int,int ) int`, `func(int,int,int)int`},

		{`func (x, y int)`, `func(x,y int)`},
		{`func (x, y int, b1, b2 byte)`, `func(x,y int,b1,b2 byte)`},
		{`func () (x, y int)`, `func()(x,y int)`},
		{`func () (x, y int, b1, b2 byte)`, `func()(x,y int,b1,b2 byte)`},

		{`func () {}`, `func(){}`},
		{`func(x int, y int) (int) { return x + y }`, `func(x int,y int)int{return x+y}`},
		{`func(x, y int) (int, int) { z := 10; return x + y, z }`, `func(x,y int)(int,int){z:=10;return x+y,z}`},
		{`func(x ...int) {}`, `func(x ...int){}`},

		{`[...]int{1, 2}`, `[...]int{1,2}`},
		{`[]int { }`, `[]int{}`},
		{`[]int { 1, }`, `[]int{1}`},
		{`[] []int{ {1}, {2,}}`, `[][]int{{1},{2}}`},
		{`map[int]int{}`, `map[int]int{}`},
		{`map[int][2] int{1: 2,}`, `map[int][2]int{1:2}`},
		{`map[ string ][]int{"a": 1, "b": 2}`, `map[string][]int{"a":1,"b":2}`},

		{`chan [ 2 ]int`, `chan [2]int`},
		{`<- chan [ 2 ]int`, `<-chan [2]int`},
		{`chan <- [ 2 ]int`, `chan<- [2]int`},

		{` f ()`, `f()`},
		{`f(1)`, `f(1)`},
		{`f(1, 2)`, `f(1,2)`},
		{`f(1, g(2, 3))`, `f(1,g(2,3))`},
		{`f( 1 )( 2, 3 )`, `f(1)(2,3)`},
		{`foo(1, args...)`, `foo(1,args...)`},

		{`struct { }`, `struct{}`},
		{`struct{ int }`, `struct{int}`},
		{`struct{ int; int }`, `struct{int;int}`},
		{`struct{int;int;int}`, `struct{int;int;int}`},
		{`struct{ Embedded; x int}`, `struct{Embedded;x int}`},

		{`struct{ x, y int }`, `struct{x,y int}`},
		{`struct{ x, y int; z int }`, `struct{x,y int;z int}`},

		{`interface{}`, `interface{}`},
		{`interface{ foo() }`, `interface{foo()}`},
		{`interface{ foo(); bar() }`, `interface{foo();bar()}`},
		{`interface{ foo(int); bar() (int, error) }`, `interface{foo(int);bar()(int,error)}`},
		{`interface { Embedded; foo() }`, `interface{Embedded;foo()}`},

		// We need a space between `<` and `-`; otherwise it'll be parsed differently.
		{`x < -y`, `x< -y`},
		{`d < -35*Second`, `d< -35*Second`},

		// We need a space between `-` and another `-`; otherwise it'll be parsed differently.
		{`x - -1`, `x- -1`},
	}

	var m minifier
	for _, test := range tests {
		expr := strparse.Expr(test.src)
		var buf bytes.Buffer
		m.Fprint(&buf, token.NewFileSet(), expr)
		have := buf.Bytes()
		if !bytes.Equal(have, []byte(test.want)) {
			t.Errorf("minify %s:\nhave: %q\nwant: %q", test.src, have, test.want)
		}
	}
}

func TestGoroot(t *testing.T) {
	var goroot string
	{
		out, err := exec.Command("go", "env", "GOROOT").CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		goroot = strings.TrimSpace(string(out))
	}

	visitFile := func(filename string) error {
		fset := token.NewFileSet()
		fileContents, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(fset, filename, fileContents, 0)
		if err != nil {
			return nil
		}
		var minified bytes.Buffer
		if err := Node(&minified, fset, f); err != nil {
			return err
		}
		fset2 := token.NewFileSet()
		f2, err := parser.ParseFile(fset2, filename, minified.Bytes(), 0)
		if err != nil {
			return fmt.Errorf("re-parse minified: %w", err)
		}
		if diff := astDiff(f, f2); diff != "" {
			return fmt.Errorf("minified code produced different AST:\n%s", diff)
		}
		return nil
	}

	srcDir := filepath.Join(goroot, "src")
	err := filepath.WalkDir(srcDir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		if err := visitFile(path); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func astDiff(x, y ast.Node) string {
	var buf bytes.Buffer
	format.Node(&buf, token.NewFileSet(), x)
	s1 := buf.String()
	buf.Reset()
	format.Node(&buf, token.NewFileSet(), y)
	s2 := buf.String()
	return cmp.Diff(s1, s2)
}
