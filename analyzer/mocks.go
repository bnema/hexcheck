package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/bnema/hexcheck/config"
	"github.com/bnema/hexcheck/internal/glob"
)

type portInterface struct {
	name  string
	iface *types.Interface
}

func (r runner) checkMissingMocks(file *ast.File, filePath string, current config.Match) {
	if current.Role != config.RolePorts {
		return
	}
	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		if _, ok := ts.Type.(*ast.InterfaceType); !ok {
			return true
		}
		if !r.generatedMockExists(ts.Name.Name) {
			r.report(ts.Name.Pos(), "missing-generated-mock-for-port", filePath, "port interface %s has no generated mock matching configured paths and name patterns", ts.Name.Name)
		}
		return true
	})
}

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

		if !isTestDoubleName(ts.Name.Name) {
			return true
		}
		obj, ok := r.pass.TypesInfo.Defs[ts.Name].(*types.TypeName)
		if !ok || obj.Type() == nil {
			return true
		}
		for _, port := range portInterfaces {
			if !r.generatedMockExists(port.name) {
				continue
			}
			if types.Implements(obj.Type(), port.iface) || types.Implements(types.NewPointer(obj.Type()), port.iface) {
				r.report(ts.Name.Pos(), "no-local-fakes-for-ports", filePath, "local test double %s implements port interface %s with a generated mock available", ts.Name.Name, port.name)
				break
			}
		}
		return true
	})
}

func (r runner) importedPortInterfaces(file *ast.File) []portInterface {
	var out []portInterface
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
				out = append(out, portInterface{name: name, iface: iface.Complete()})
			}
		}
	}
	return out
}

var mockIndexCache sync.Map

func (r runner) generatedMockExists(interfaceName string) bool {
	if r.cfg.Root == "" || len(r.cfg.Mocking.GeneratedMockPaths) == 0 {
		return false
	}
	index := cachedMockIndex(r.cfg.Root, r.cfg.Mocking.GeneratedMockPaths)
	for _, name := range r.expectedMockNames(interfaceName) {
		if index[name] {
			return true
		}
	}
	return false
}

func cachedMockIndex(root string, patterns []string) map[string]bool {
	key := mockIndexKey(root, patterns)
	if cached, ok := mockIndexCache.Load(key); ok {
		return cached.(map[string]bool)
	}
	index := buildMockIndex(root, patterns)
	actual, _ := mockIndexCache.LoadOrStore(key, index)
	return actual.(map[string]bool)
}

func mockIndexKey(root string, patterns []string) string {
	patterns = append([]string(nil), patterns...)
	sort.Strings(patterns)
	return root + "\x00" + strings.Join(patterns, "\x00")
}

func buildMockIndex(root string, patterns []string) map[string]bool {
	index := map[string]bool{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if !matchesAnyGlob(patterns, rel) {
			return nil
		}
		for _, name := range declaredTypes(path) {
			index[name] = true
		}
		return nil
	})
	return index
}

func declaredTypes(path string) []string {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil
	}
	var names []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if ok {
				names = append(names, ts.Name.Name)
			}
		}
	}
	return names
}

func (r runner) expectedMockNames(interfaceName string) []string {
	patterns := r.cfg.Mocking.GeneratedMockNamePatterns
	if len(patterns) == 0 {
		patterns = []string{"Mock{{Interface}}", "{{Interface}}Mock"}
	}
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		out = append(out, strings.ReplaceAll(pattern, "{{Interface}}", interfaceName))
	}
	return out
}

func matchesAnyGlob(patterns []string, path string) bool {
	for _, pattern := range patterns {
		if glob.Match(pattern, path) {
			return true
		}
	}
	return false
}

func isTestDoubleName(name string) bool {
	return hasWordPrefix(name, "fake") || hasWordSuffix(name, "fake") || hasWordPrefix(name, "stub") || hasWordSuffix(name, "stub")
}

func hasWordPrefix(name, prefix string) bool {
	if len(name) >= len(prefix) && strings.EqualFold(name[:len(prefix)], prefix) && len(name) == len(prefix) {
		return true
	}
	if len(name) < len(prefix) || !strings.EqualFold(name[:len(prefix)], prefix) {
		return false
	}
	return startsCamelWord(name[len(prefix):])
}

func hasWordSuffix(name, suffix string) bool {
	if len(name) < len(suffix) || !strings.EqualFold(name[len(name)-len(suffix):], suffix) {
		return false
	}
	prefix := name[:len(name)-len(suffix)]
	return prefix == "" || strings.HasSuffix(prefix, "_")
}

func startsCamelWord(s string) bool {
	if s == "" {
		return true
	}
	r := rune(s[0])
	return r == '_' || (r >= 'A' && r <= 'Z')
}
