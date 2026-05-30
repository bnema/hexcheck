package analyzer

import (
	"go/ast"
	"strconv"
	"strings"

	"github.com/bnema/hexcheck/config"
)

func (r runner) checkMocks(file *ast.File, filePath string, current config.Match) {
	if !strings.HasSuffix(filePath, "_test.go") {
		return
	}

	if current.Role == config.RoleUsecase || current.Role == config.RoleCore {
		for _, imp := range file.Imports {
			importPath, _ := strconv.Unquote(imp.Path.Value)
			if match, ok := r.cfg.ComponentForPath(r.relImportPath(importPath)); ok && match.Role == config.RoleAdapter {
				r.report(imp.Pos(), "prefer-generated-mocks", filePath, "test imports concrete adapter %s; prefer generated mocks for port dependencies", importPath)
			}
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		name := strings.ToLower(ts.Name.Name)
		if strings.HasPrefix(name, "fake") || strings.HasSuffix(name, "fake") || strings.HasPrefix(name, "stub") || strings.HasSuffix(name, "stub") {
			r.report(ts.Name.Pos(), "no-local-fakes-for-ports", filePath, "local test double %s declared in test file; prefer generated mocks when a port interface is available", ts.Name.Name)
		}
		return true
	})
}
