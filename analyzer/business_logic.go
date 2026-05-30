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
	if r.cfg.Heuristics.ExcludeTestFiles != nil && *r.cfg.Heuristics.ExcludeTestFiles && strings.HasSuffix(filePath, "_test.go") {
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
			if containsWord(fn.Name.Name, keyword) {
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

func containsWord(name, word string) bool {
	if word == "" {
		return false
	}
	for i := 0; i <= len(name)-len(word); i++ {
		if !strings.EqualFold(name[i:i+len(word)], word) {
			continue
		}
		beforeOK := i == 0 || isBoundaryRune(rune(name[i-1]), rune(name[i]))
		after := i + len(word)
		afterOK := after == len(name) || isBoundaryRune(rune(name[after-1]), rune(name[after]))
		if beforeOK && afterOK {
			return true
		}
	}
	return false
}

func isBoundaryRune(prev, next rune) bool {
	return prev == '_' || next == '_' || (prev >= 'a' && prev <= 'z' && next >= 'A' && next <= 'Z')
}
