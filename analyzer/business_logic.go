package analyzer

import (
	"go/ast"
	"strings"

	"github.com/bnema/hexcheck/config"
)

func (r runner) checkBusinessLogic(file *ast.File, filePath string, current config.Match) {
	if current.Role != config.RoleAdapter && current.Role != config.RoleEntrypoint {
		return
	}

	threshold := r.cfg.Heuristics.BusinessLogicThreshold
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		score := 0
		for _, keyword := range r.cfg.Heuristics.BusinessKeywords {
			if strings.Contains(fn.Name.Name, keyword) {
				score += 2
			}
		}

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n.(type) {
			case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.ForStmt, *ast.RangeStmt:
				score++
			}
			return true
		})

		if score >= threshold {
			r.report(fn.Name.Pos(), "suspicious-business-logic-in-adapter", filePath, "adapter function %s has suspicious business-logic score %d", fn.Name.Name, score)
		}
	}
}
