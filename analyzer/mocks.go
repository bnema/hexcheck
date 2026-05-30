package analyzer

import (
	"go/ast"
	"go/types"
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

	portInterfaces := r.importedPortInterfaces(file)
	if len(portInterfaces) == 0 {
		return
	}

	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		name := strings.ToLower(ts.Name.Name)
		if !isTestDoubleName(name) {
			return true
		}
		obj, ok := r.pass.TypesInfo.Defs[ts.Name].(*types.TypeName)
		if !ok || obj.Type() == nil {
			return true
		}
		for _, iface := range portInterfaces {
			if types.Implements(obj.Type(), iface) || types.Implements(types.NewPointer(obj.Type()), iface) {
				r.report(ts.Name.Pos(), "no-local-fakes-for-ports", filePath, "local test double %s implements port interface %s; prefer generated mocks when available", ts.Name.Name, iface.String())
				break
			}
		}
		return true
	})
}

func (r runner) importedPortInterfaces(file *ast.File) []*types.Interface {
	var out []*types.Interface
	for _, imp := range file.Imports {
		pkgName, ok := r.pass.TypesInfo.Implicits[imp].(*types.PkgName)
		if !ok || pkgName.Imported() == nil {
			continue
		}
		if match, ok := r.cfg.ComponentForPath(r.relImportPath(pkgName.Imported().Path())); !ok || match.Role != config.RolePorts {
			continue
		}
		scope := pkgName.Imported().Scope()
		for _, name := range scope.Names() {
			obj, ok := scope.Lookup(name).(*types.TypeName)
			if !ok {
				continue
			}
			iface, ok := obj.Type().Underlying().(*types.Interface)
			if ok {
				out = append(out, iface.Complete())
			}
		}
	}
	return out
}

func isTestDoubleName(name string) bool {
	return strings.HasPrefix(name, "fake") || strings.HasSuffix(name, "fake") || strings.HasPrefix(name, "stub") || strings.HasSuffix(name, "stub")
}
