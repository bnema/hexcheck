package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/bnema/hexcheck/config"
	"github.com/bnema/hexcheck/internal/glob"
	"golang.org/x/tools/go/analysis"
)

const Name = "hexcheck"

type Options struct {
	Config     *config.Config
	ConfigPath string
	Root       string
	ModulePath string
}

// Result is returned for each analyzed package. Drivers use it to map a
// diagnostic Category (the rule name) to its configured severity.
type Result struct {
	Severities map[string]config.Severity
}

// Severity returns the configured severity of a diagnostic category.
func (r *Result) Severity(rule string) config.Severity {
	if r == nil {
		return config.SeverityOff
	}
	return r.Severities[rule]
}

// state is shared by every package analyzed by one Analyzer instance, so the
// config and mock index are loaded once per run instead of once per package.
type state struct {
	once       sync.Once
	cfg        *config.Config
	modulePath string
	err        error
	mockOnce   sync.Once
	mocks      map[string]bool
}

func New(opts Options) *analysis.Analyzer {
	var configPathFlag string
	var rootFlag string
	var modulePathFlag string
	st := &state{}

	a := &analysis.Analyzer{
		Name:       Name,
		Doc:        "checks hexagonal architecture boundaries",
		ResultType: reflect.TypeFor[*Result](),
		Run: func(pass *analysis.Pass) (any, error) {
			st.once.Do(func() {
				cfg, root, err := resolveConfig(opts, configPathFlag, rootFlag)
				if err != nil {
					st.err = err
					return
				}
				modulePath := firstNonEmpty(opts.ModulePath, modulePathFlag)
				if modulePath == "" {
					modulePath = discoverModulePath(root)
				}
				st.cfg, st.modulePath = cfg, modulePath
			})
			if st.err != nil {
				return nil, st.err
			}
			r := runner{pass: pass, cfg: st.cfg, modulePath: st.modulePath, state: st}
			r.run()
			return &Result{Severities: st.cfg.Rules}, nil
		},
	}
	if opts.Config == nil {
		a.Flags.StringVar(&configPathFlag, "config", opts.ConfigPath, "path to .hexcheck.yaml")
		a.Flags.StringVar(&rootFlag, "root", opts.Root, "project root for config-relative paths")
		a.Flags.StringVar(&modulePathFlag, "module", opts.ModulePath, "Go module path; defaults to module in go.mod")
	}
	return a
}

func resolveConfig(opts Options, configPathFlag, rootFlag string) (*config.Config, string, error) {
	if opts.Config != nil {
		cfg := cloneConfig(opts.Config)
		cfg.ApplyDefaults()
		if err := cfg.Validate(); err != nil {
			return nil, "", err
		}
		if cfg.Root != "" {
			root, err := filepath.Abs(cfg.Root)
			if err != nil {
				return nil, "", err
			}
			cfg.Root = root
		}
		return cfg, cfg.Root, nil
	}
	root := firstNonEmpty(opts.Root, rootFlag)
	if root != "" {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, "", err
		}
		root = absRoot
	}
	configPath := firstNonEmpty(opts.ConfigPath, configPathFlag)
	if configPath == "" {
		start := root
		if start == "" {
			start = "."
		}
		discovered, err := config.DiscoverConfig(start)
		if err == nil {
			configPath = discovered
		}
	}
	if configPath == "" {
		cfg := config.Default()
		if root != "" {
			cfg.Root = root
		}
		return cfg, cfg.Root, nil
	}
	if root != "" && !filepath.IsAbs(configPath) {
		configPath = filepath.Join(root, configPath)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, "", err
	}
	if root != "" {
		cfg.Root = root
	}
	return cfg, cfg.Root, nil
}

func discoverModulePath(root string) string {
	if root == "" {
		root = "."
	}
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if modulePath, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(modulePath)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type runner struct {
	pass       *analysis.Pass
	cfg        *config.Config
	modulePath string
	state      *state
	ignores    ignoreIndex
}

func (r runner) run() {
	pkgRel := r.relImportPath(r.pass.Pkg.Path())
	current, ok := r.cfg.ComponentForPath(pkgRel)
	if !ok || current.Role == config.RoleIgnore {
		return
	}
	r.ignores = ignoreIndex{}
	businessDiagnostics := 0
	for _, file := range r.pass.Files {
		filePath := r.filePath(file)
		if isGeneratedFile(file) {
			continue
		}
		r.collectIgnores(file)
		r.checkImports(file, filePath, current)
		r.checkTypeLeaks(file, filePath, current)
		r.checkMissingMocks(file, filePath, current)
		r.checkBusinessLogic(file, filePath, current, &businessDiagnostics)
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
		case current.Role == config.RoleCore && imported.Role == config.RoleUsecase:
			r.report(imp.Pos(), "no-usecase-imports-in-core", filePath, "core component %q imports usecase component %q (%s)", current.Name, imported.Name, importPath)
		case current.Role == config.RoleUsecase && (imported.Role == config.RoleAdapter || imported.Role == config.RoleEntrypoint):
			r.report(imp.Pos(), "no-infra-imports-in-usecase", filePath, "usecase component %q imports %s component %q (%s)", current.Name, imported.Role, imported.Name, importPath)
		case current.Role == config.RolePorts && (imported.Role == config.RoleAdapter || imported.Role == config.RoleEntrypoint):
			r.report(imp.Pos(), "no-infra-imports-in-ports", filePath, "ports component %q imports %s component %q (%s)", current.Name, imported.Role, imported.Name, importPath)
		case current.Role == config.RoleAdapter && imported.Role == config.RoleEntrypoint:
			r.report(imp.Pos(), "no-entrypoint-imports-in-adapter", filePath, "adapter component %q imports entrypoint component %q (%s)", current.Name, imported.Name, importPath)
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
		t := r.pass.TypesInfo.TypeOf(field.Type)
		if r.isInfraType(t, nil) {
			r.report(field.Pos(), rule, filePath, "exposes infrastructure/framework type %s", t)
		}
	}
}

// isInfraType reports whether t, or any type it is composed of, belongs to an
// adapter/entrypoint component or a configured external framework package.
func (r runner) isInfraType(t types.Type, seen map[types.Type]bool) bool {
	if t == nil {
		return false
	}
	if seen[t] {
		return false
	}
	if seen == nil {
		seen = map[types.Type]bool{}
	}
	seen[t] = true
	switch tt := t.(type) {
	case *types.Alias:
		return r.isInfraType(types.Unalias(tt), seen)
	case *types.Pointer:
		return r.isInfraType(tt.Elem(), seen)
	case *types.Slice:
		return r.isInfraType(tt.Elem(), seen)
	case *types.Array:
		return r.isInfraType(tt.Elem(), seen)
	case *types.Chan:
		return r.isInfraType(tt.Elem(), seen)
	case *types.Map:
		return r.isInfraType(tt.Key(), seen) || r.isInfraType(tt.Elem(), seen)
	case *types.Signature:
		return r.isInfraType(tt.Params(), seen) || r.isInfraType(tt.Results(), seen)
	case *types.Tuple:
		for v := range tt.Variables() {
			if r.isInfraType(v.Type(), seen) {
				return true
			}
		}
		return false
	case *types.Struct:
		for field := range tt.Fields() {
			if r.isInfraType(field.Type(), seen) {
				return true
			}
		}
		return false
	case *types.Named:
		for arg := range tt.TypeArgs().Types() {
			if r.isInfraType(arg, seen) {
				return true
			}
		}
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
		if glob.Match(pattern, pkg) || glob.Match(pattern+"/**", pkg) {
			return true
		}
	}
	return false
}

// report emits a diagnostic for rule unless it is disabled, allowed by config,
// excluded by path, or suppressed by an inline ignore directive.
func (r runner) report(pos token.Pos, rule, filePath, format string, args ...any) bool {
	if !r.cfg.RuleEnabled(rule) || r.cfg.IsAllowed(rule, filePath) || r.isExcluded(rule, filePath) || r.isIgnoredInline(pos, rule) {
		return false
	}
	r.pass.Report(analysis.Diagnostic{
		Pos:      pos,
		Category: rule,
		Message:  fmt.Sprintf("[%s] %s: %s", r.cfg.Severity(rule), rule, fmt.Sprintf(format, args...)),
	})
	return true
}

func (r runner) isExcluded(rule, filePath string) bool {
	setting, ok := r.cfg.RuleSettings[rule]
	if !ok {
		return false
	}
	for _, pattern := range setting.ExcludePaths {
		if glob.Match(pattern, filePath) {
			return true
		}
	}
	return false
}

func (r runner) relImportPath(importPath string) string {
	if r.modulePath != "" {
		if rel, ok := strings.CutPrefix(importPath, r.modulePath+"/"); ok {
			return rel
		}
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
		if rel, ok := strings.CutPrefix(pos, root+"/"); ok {
			return rel
		}
	}
	return pos
}
