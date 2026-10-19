package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/bnema/hexcheck/config"
	"golang.org/x/tools/go/analysis"
)

func TestOversizedFunctionSkipsTypeAndLocalCollection(t *testing.T) {
	fn := &ast.FuncDecl{Body: &ast.BlockStmt{List: []ast.Stmt{
		&ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent("repo"), Sel: ast.NewIdent("Name")}}, Rhs: []ast.Expr{ast.NewIdent("value")}},
	}}}
	// No analysis pass: any type query before rejection would panic.
	r := runner{cfg: &config.Config{Heuristics: config.Heuristics{BusinessLogicMaxFunctionNodes: testIntPtr(1)}}}
	facts := r.collectBusinessFunctionFacts(fn, config.RoleAdapter)
	if facts.nodeCount <= 1 || facts.constructedLocals != nil || facts.domainMutation {
		t.Fatalf("oversized function was analyzed: %+v", facts)
	}
}

func TestConstructionFactsCachedByObject(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "locals.go", `package p
 type Value struct { N int }
 func fetch() Value { return Value{} }
 func build(param Value) {
  fresh := Value{}
  fresh.N = 1
  var zero Value
  zero.N = 2
  replaced := Value{}
  replaced = fetch()
  replaced.N = 3
  fetched := fetch()
  fetched.N = 4
  param.N = 5
  _ = fresh; _ = zero; _ = replaced; _ = fetched
 }`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}}
	if _, err := new(types.Config).Check("p", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[2].(*ast.FuncDecl)
	facts := businessFunctionFacts{fn: fn, runner: runner{pass: &analysis.Pass{TypesInfo: info}}}
	facts.collectLocals()
	want := map[string]bool{"fresh": true, "zero": true, "replaced": false, "fetched": false, "param": false}
	checked := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			id := sel.X.(*ast.Ident)
			if got := facts.isConstructedLocal(sel); got != want[id.Name] {
				t.Errorf("%s constructed=%v, want %v", id.Name, got, want[id.Name])
			}
			checked++
		}
		return true
	})
	if checked != len(want) {
		t.Fatalf("checked %d receivers, want %d", checked, len(want))
	}
}

func TestCollectBusinessFunctionFactsInspectsFunctionBody(t *testing.T) {
	fn := &ast.FuncDecl{
		Name: ast.NewIdent("ResolveMode"),
		Type: &ast.FuncType{},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.IfStmt{
				Cond: ast.NewIdent("ok"),
				Body: &ast.BlockStmt{},
			},
		}},
	}
	r := runner{
		cfg:  &config.Config{Heuristics: config.Heuristics{BusinessLogicMaxFunctionNodes: testIntPtr(2000)}},
		pass: &analysis.Pass{TypesInfo: &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}},
	}

	facts := r.collectBusinessFunctionFacts(fn, config.RoleAdapter)

	if facts.fn != fn {
		t.Fatalf("collectBusinessFunctionFacts() fn = %p, want %p", facts.fn, fn)
	}
	if facts.branchCount != 1 {
		t.Fatalf("collectBusinessFunctionFacts() branchCount = %d, want 1", facts.branchCount)
	}
	if facts.nodeCount == 0 {
		t.Fatal("collectBusinessFunctionFacts() nodeCount = 0, want inspected nodes")
	}
}
