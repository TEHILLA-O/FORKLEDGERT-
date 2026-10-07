package patches

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

func ParseGoFunctions(src, filename string) []string {
	if !strings.HasSuffix(strings.ToLower(filename), ".go") {
		return nil
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Base(filename), src, 0)
	if err != nil {
		return nil
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			return true
		}
		name := fn.Name.Name
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			name = recvName(fn.Recv.List[0].Type) + "." + name
		}
		out = append(out, name)
		return true
	})
	return unique(out)
}

func recvName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return recvName(t.X)
	case *ast.IndexExpr:
		return recvName(t.X)
	default:
		return "T"
	}
}
