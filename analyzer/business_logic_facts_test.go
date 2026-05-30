package analyzer

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/bnema/hexcheck/config"
	"golang.org/x/tools/go/analysis"
)

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
