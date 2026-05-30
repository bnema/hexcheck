package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/bnema/hexcheck/config"
	"golang.org/x/tools/go/analysis"
)

const Name = "hexcheck"

type Options struct {
	Config     *config.Config
	ModulePath string
}

func New(opts Options) *analysis.Analyzer {
	cfg := opts.Config
	if cfg == nil {
		cfg = config.Default()
	}
	return &analysis.Analyzer{
		Name: Name,
		Doc:  "checks hexagonal architecture boundaries",
		Run: func(pass *analysis.Pass) (any, error) {
			r := runner{pass: pass, cfg: cfg, modulePath: opts.ModulePath}
			r.run()
			return nil, nil
		},
	}
}

type runner struct {
	pass       *analysis.Pass
	cfg        *config.Config
	modulePath string
}

func (r runner) run() {
	pkgRel := r.relImportPath(r.pass.Pkg.Path())
	current, ok := r.cfg.ComponentForPath(pkgRel)
	if !ok || current.Role == config.RoleIgnore {
		return
	}
	for _, file := range r.pass.Files {
		filePath := r.filePath(file)
		r.checkImports(file, filePath, current)
		r.checkTypeLeaks(file, filePath, current)
		r.checkBusinessLogic(file, filePath, current)
		r.checkMocks(file, filePath, current)
	}
}

func (r runner) checkImports(file *ast.File, filePath string, current config.Match) {
	for _, imp := range file.Imports {
		importPath, _ := strconv.Unquote(imp.Path.Value)
		importRel := r.relImportPath(importPath)
		imported, ok := r.cfg.ComponentForPath(importRel)
		if !ok || imported.Role == config.RoleIgnore {
			continue
		}
		switch {
		case current.Role == config.RoleCore && (imported.Role == config.RoleAdapter || imported.Role == config.RoleEntrypoint):
			r.report(imp.Pos(), "no-adapter-imports-in-core", filePath, "core component %q imports %s component %q (%s)", current.Name, imported.Role, imported.Name, importPath)
		case current.Role == config.RoleUsecase && imported.Role == config.RoleAdapter:
			r.report(imp.Pos(), "no-infra-imports-in-usecase", filePath, "usecase component %q imports adapter component %q (%s)", current.Name, imported.Name, importPath)
		case current.Role == config.RoleAdapter && imported.Role == config.RoleAdapter && current.Name != imported.Name:
			r.report(imp.Pos(), "no-adapter-to-adapter-imports", filePath, "adapter component %q imports adapter component %q (%s)", current.Name, imported.Name, importPath)
		}
	}
}

func (r runner) checkTypeLeaks(file *ast.File, filePath string, current config.Match) {
	if current.Role != config.RolePorts && current.Role != config.RoleCore {
		return
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if current.Role == config.RoleCore && node.Name.IsExported() {
				r.checkFieldList(node.Type.Params, filePath, "no-framework-types-in-core")
				r.checkFieldList(node.Type.Results, filePath, "no-framework-types-in-core")
			}
		case *ast.TypeSpec:
			if current.Role == config.RolePorts {
				if iface, ok := node.Type.(*ast.InterfaceType); ok {
					for _, method := range iface.Methods.List {
						if fn, ok := method.Type.(*ast.FuncType); ok {
							r.checkFieldList(fn.Params, filePath, "no-infra-types-in-ports")
							r.checkFieldList(fn.Results, filePath, "no-infra-types-in-ports")
						}
					}
				}
			}
			if current.Role == config.RoleCore && node.Name.IsExported() {
				if st, ok := node.Type.(*ast.StructType); ok {
					r.checkFieldList(st.Fields, filePath, "no-framework-types-in-core")
				}
			}
		}
		return true
	})
}

func (r runner) checkFieldList(fields *ast.FieldList, filePath, rule string) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		if r.isInfraType(r.pass.TypesInfo.TypeOf(field.Type)) {
			r.report(field.Pos(), rule, filePath, "%s exposes infrastructure/framework type %s", rule, r.pass.TypesInfo.TypeOf(field.Type))
		}
	}
}

func (r runner) isInfraType(t types.Type) bool {
	if t == nil {
		return false
	}
	switch tt := t.(type) {
	case *types.Pointer:
		return r.isInfraType(tt.Elem())
	case *types.Slice:
		return r.isInfraType(tt.Elem())
	case *types.Array:
		return r.isInfraType(tt.Elem())
	case *types.Map:
		return r.isInfraType(tt.Key()) || r.isInfraType(tt.Elem())
	case *types.Named:
		obj := tt.Obj()
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		pkgPath := obj.Pkg().Path()
		rel := r.relImportPath(pkgPath)
		if match, ok := r.cfg.ComponentForPath(rel); ok && (match.Role == config.RoleAdapter || match.Role == config.RoleEntrypoint) {
			return true
		}
		return r.matchesExternal(pkgPath) || r.matchesExternal(rel)
	default:
		return false
	}
}

func (r runner) matchesExternal(pkg string) bool {
	for _, pattern := range append(r.cfg.ExternalTypes.FrameworkPackages, r.cfg.ExternalTypes.AdapterTypePackages...) {
		if simpleMatch(pattern, pkg) {
			return true
		}
	}
	return false
}

func (r runner) report(pos token.Pos, rule, filePath, format string, args ...any) {
	if !r.cfg.RuleEnabled(rule) || r.cfg.IsAllowed(rule, filePath) {
		return
	}
	r.pass.Reportf(pos, rule+": "+format, args...)
}

func (r runner) relImportPath(importPath string) string {
	if r.modulePath != "" && strings.HasPrefix(importPath, r.modulePath+"/") {
		return strings.TrimPrefix(importPath, r.modulePath+"/")
	}
	if r.modulePath != "" && importPath == r.modulePath {
		return ""
	}
	return importPath
}

func (r runner) filePath(file *ast.File) string {
	pos := r.pass.Fset.Position(file.Pos()).Filename
	pos = strings.ReplaceAll(pos, "\\", "/")
	if idx := strings.LastIndex(pos, "/src/"+r.modulePath+"/"); idx >= 0 && r.modulePath != "" {
		return pos[idx+len("/src/")+len(r.modulePath)+1:]
	}
	if r.cfg.Root != "" {
		root := strings.ReplaceAll(r.cfg.Root, "\\", "/")
		if strings.HasPrefix(pos, root+"/") {
			return strings.TrimPrefix(pos, root+"/")
		}
	}
	return pos
}

func simpleMatch(pattern, name string) bool {
	pattern = strings.TrimSuffix(pattern, "/**")
	return name == pattern || strings.HasPrefix(name, pattern+"/")
}
