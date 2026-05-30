package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/bnema/hexcheck/config"
)

type businessLogicConfidence string

const (
	businessLogicConfidenceNone   businessLogicConfidence = "none"
	businessLogicConfidenceLow    businessLogicConfidence = "low"
	businessLogicConfidenceMedium businessLogicConfidence = "medium"
	businessLogicConfidenceHigh   businessLogicConfidence = "high"
)

type businessLogicSignal struct {
	kind   string
	strong bool
}

func (r runner) checkBusinessLogic(file *ast.File, filePath string, current config.Match, packageDiagnostics *int) {
	if current.Role != config.RoleAdapter && current.Role != config.RoleEntrypoint {
		return
	}
	if r.cfg.Heuristics.ExcludeTestFiles != nil && *r.cfg.Heuristics.ExcludeTestFiles && strings.HasSuffix(filePath, "_test.go") {
		return
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if *packageDiagnostics >= *r.cfg.Heuristics.BusinessLogicMaxDiagnosticsPerPackage {
			return
		}
		if r.isSuspiciousBusinessFunction(fn, file, filePath, current.Role) {
			*packageDiagnostics++
		}
	}
}

func (r runner) isSuspiciousBusinessFunction(fn *ast.FuncDecl, file *ast.File, filePath string, fileRole config.Role) bool {
	facts := r.collectBusinessFunctionFacts(fn, fileRole, file)
	if facts.nodeCount > *r.cfg.Heuristics.BusinessLogicMaxFunctionNodes {
		return false
	}
	strong, weak := facts.classify()
	if facts.isTechnicalDetectionOnly() {
		return false
	}
	confidence := facts.businessLogicConfidence(strong, weak)
	if confidenceMeets(confidence, effectiveBusinessLogicMinConfidence(r.cfg)) {
		return r.report(fn.Name.Pos(), "suspicious-business-logic-in-adapter", filePath, "%s function %s has business-logic evidence: confidence=%s strong=%v weak=%v", facts.componentRole(), fn.Name.Name, confidence, signalKinds(strong), signalKinds(weak))
	}
	return false
}

func (r runner) collectBusinessFunctionFacts(fn *ast.FuncDecl, fileRole config.Role, file ...*ast.File) businessFunctionFacts {
	if fileRole == "" {
		fileRole = config.RoleAdapter
	}
	facts := businessFunctionFacts{runner: r, fn: fn, fileRole: fileRole, imports: map[string]bool{}}
	if len(file) > 0 && file[0] != nil {
		for _, imp := range file[0].Imports {
			facts.imports[strings.Trim(imp.Path.Value, "\"")] = true
		}
	}
	ast.Inspect(fn.Body, facts.inspect)
	return facts
}

type businessFunctionFacts struct {
	runner   runner
	fn       *ast.FuncDecl
	fileRole config.Role

	nodeCount             int
	branchCount           int
	keyword               bool
	domainTypeUse         bool
	domainMutation        bool
	domainMethodCall      bool
	domainConstructorCall bool
	domainErrorReturn     bool
	policyConstant        bool
	usecaseCalls          map[string]bool
	imports               map[string]bool
	usesTechnicalPackage  bool
}

func (f *businessFunctionFacts) inspect(n ast.Node) bool {
	if n == nil {
		return true
	}
	f.nodeCount++
	if f.nodeCount > *f.runner.cfg.Heuristics.BusinessLogicMaxFunctionNodes {
		return false
	}

	if f.exprUsesRole(exprFromNode(n), config.RoleCore) || f.exprUsesRole(exprFromNode(n), config.RoleUsecase) {
		f.domainTypeUse = true
	}

	switch node := n.(type) {
	case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.ForStmt, *ast.RangeStmt:
		f.branchCount++
	case *ast.AssignStmt:
		for _, lhs := range node.Lhs {
			if f.selectorReceiverUsesRole(lhs, config.RoleCore, config.RoleUsecase) {
				f.domainMutation = true
			}
		}
	case *ast.CallExpr:
		f.inspectCall(node)
	case *ast.BasicLit:
		if node.Kind == token.INT || node.Kind == token.FLOAT || node.Kind == token.STRING {
			f.policyConstant = true
		}
	case *ast.ReturnStmt:
		for _, result := range node.Results {
			if f.exprUsesRole(result, config.RoleCore) || f.exprUsesRole(result, config.RoleUsecase) {
				f.domainErrorReturn = true
			}
		}
	}
	return true
}

func (f *businessFunctionFacts) inspectCall(call *ast.CallExpr) {
	f.inspectTechnicalCall(call)
	calleeType := f.runner.pass.TypesInfo.TypeOf(call.Fun)
	if f.typeHasRole(calleeType, config.RoleCore) || f.typeHasRole(calleeType, config.RoleUsecase) {
		f.domainConstructorCall = true
	}
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if f.exprUsesRole(sel.X, config.RoleCore) {
			f.domainMethodCall = true
		}
		if f.exprUsesRole(sel.X, config.RolePorts) || f.exprUsesRole(sel.X, config.RoleUsecase) {
			if f.usecaseCalls == nil {
				f.usecaseCalls = map[string]bool{}
			}
			f.usecaseCalls[collaboratorKey(f.runner.pass.TypesInfo, sel.X)] = true
		}
	}
	for _, arg := range call.Args {
		if f.exprUsesRole(arg, config.RoleCore) || f.exprUsesRole(arg, config.RoleUsecase) {
			f.domainTypeUse = true
		}
	}
}

func (f *businessFunctionFacts) inspectTechnicalCall(call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok || f.runner.pass == nil || f.runner.pass.TypesInfo == nil {
		return
	}
	pkgName, ok := f.runner.pass.TypesInfo.ObjectOf(ident).(*types.PkgName)
	if !ok || pkgName.Imported() == nil {
		return
	}
	path := pkgName.Imported().Path()
	if technicalImportPaths[path] && f.imports[path] {
		f.usesTechnicalPackage = true
	}
}

var technicalImportPaths = map[string]bool{
	"os":            true,
	"path/filepath": true,
	"strconv":       true,
	"bufio":         true,
}

func collaboratorKey(info *types.Info, expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		if obj := info.ObjectOf(ident); obj != nil {
			return obj.Pkg().Path() + "." + obj.Name()
		}
	}
	return fmt.Sprintf("%p", expr)
}

func (f *businessFunctionFacts) businessLogicConfidence(strong, weak []businessLogicSignal) businessLogicConfidence {
	if len(strong) >= *f.runner.cfg.Heuristics.BusinessLogicMinStrongSignals {
		return businessLogicConfidenceHigh
	}
	if len(strong) >= 1 && len(weak) >= *f.runner.cfg.Heuristics.BusinessLogicMinWeakSignals {
		return businessLogicConfidenceMedium
	}
	if len(strong) >= 1 || len(weak) >= *f.runner.cfg.Heuristics.BusinessLogicMinWeakSignals {
		return businessLogicConfidenceLow
	}
	return businessLogicConfidenceNone
}

func effectiveBusinessLogicMinConfidence(cfg *config.Config) string {
	min := cfg.Heuristics.BusinessLogicMinConfidence
	if cfg.Heuristics.BusinessLogicMode == "ci" && confidenceMeets(businessLogicConfidenceHigh, min) {
		return string(businessLogicConfidenceHigh)
	}
	return min
}

func confidenceMeets(got businessLogicConfidence, min string) bool {
	levels := map[businessLogicConfidence]int{
		businessLogicConfidenceNone:   0,
		businessLogicConfidenceLow:    1,
		businessLogicConfidenceMedium: 2,
		businessLogicConfidenceHigh:   3,
	}
	want, ok := levels[businessLogicConfidence(min)]
	if !ok {
		want = levels[businessLogicConfidenceLow]
	}
	return levels[got] >= want
}

func (f *businessFunctionFacts) isTechnicalDetectionOnly() bool {
	if !containsWord(f.fn.Name.Name, "Detect") {
		return false
	}
	if !f.usesTechnicalPackage {
		return false
	}
	return !f.domainTypeUse && !f.domainMutation && !f.domainMethodCall && !f.domainConstructorCall && !f.domainErrorReturn && len(f.usecaseCalls) == 0
}

func (f *businessFunctionFacts) componentRole() config.Role {
	return f.fileRole
}

func (f *businessFunctionFacts) classify() ([]businessLogicSignal, []businessLogicSignal) {
	strong := []businessLogicSignal{}
	weak := []businessLogicSignal{}
	if len(f.fn.Type.Params.List) > 0 {
		for _, field := range f.fn.Type.Params.List {
			if f.typeHasAnyRole(f.runner.pass.TypesInfo.TypeOf(field.Type), config.RoleCore, config.RoleUsecase) {
				f.domainTypeUse = true
			}
		}
	}
	for _, keyword := range f.runner.cfg.Heuristics.BusinessKeywords {
		if containsWord(f.fn.Name.Name, keyword) {
			f.keyword = true
			break
		}
	}

	if f.domainMutation {
		strong = append(strong, businessLogicSignal{kind: "domain mutation", strong: true})
	}
	if f.domainMethodCall {
		strong = append(strong, businessLogicSignal{kind: "domain method call", strong: true})
	}
	if f.keyword && len(f.usecaseCalls) >= 2 {
		strong = append(strong, businessLogicSignal{kind: "business keyword with multiple port/usecase collaborators", strong: true})
	}
	if f.keyword && f.branchCount >= 3 && f.policyConstant && policyKeyword(f.fn.Name.Name) {
		strong = append(strong, businessLogicSignal{kind: "business keyword with policy constants and branching", strong: true})
	}
	if f.keyword && (f.domainTypeUse || f.domainMethodCall || f.domainMutation || f.domainConstructorCall) {
		strong = append(strong, businessLogicSignal{kind: "business keyword with domain context", strong: true})
	}

	if f.keyword {
		weak = append(weak, businessLogicSignal{kind: "business keyword", strong: false})
	}
	if f.policyConstant {
		weak = append(weak, businessLogicSignal{kind: "policy constants", strong: false})
	}
	if f.domainErrorReturn {
		weak = append(weak, businessLogicSignal{kind: "domain result return", strong: false})
	}
	if len(f.usecaseCalls) >= 2 {
		weak = append(weak, businessLogicSignal{kind: "multiple port/usecase collaborators", strong: false})
	}
	if f.branchCount >= *f.runner.cfg.Heuristics.BusinessLogicThreshold {
		weak = append(weak, businessLogicSignal{kind: fmt.Sprintf("branch count %d", f.branchCount), strong: false})
	}
	return strong, weak
}

func policyKeyword(name string) bool {
	for _, word := range []string{"Detect", "Migrate", "Resolve", "Profile", "Score", "Ranking", "Restore", "Purge", "Performance", "Selected"} {
		if containsWord(name, word) {
			return true
		}
	}
	return false
}

func (f *businessFunctionFacts) exprUsesRole(expr ast.Expr, roles ...config.Role) bool {
	if expr == nil {
		return false
	}
	return f.typeHasAnyRole(f.runner.pass.TypesInfo.TypeOf(expr), roles...)
}

func (f *businessFunctionFacts) selectorReceiverUsesRole(expr ast.Expr, roles ...config.Role) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return f.exprUsesRole(sel.X, roles...)
}

func (f *businessFunctionFacts) typeHasAnyRole(t types.Type, roles ...config.Role) bool {
	for _, role := range roles {
		if f.typeHasRole(t, role) {
			return true
		}
	}
	return false
}

func (f *businessFunctionFacts) typeHasRole(t types.Type, role config.Role) bool {
	if t == nil {
		return false
	}
	switch tt := t.(type) {
	case *types.Pointer:
		return f.typeHasRole(tt.Elem(), role)
	case *types.Slice:
		return f.typeHasRole(tt.Elem(), role)
	case *types.Array:
		return f.typeHasRole(tt.Elem(), role)
	case *types.Map:
		return f.typeHasRole(tt.Key(), role) || f.typeHasRole(tt.Elem(), role)
	case *types.Signature:
		return f.typeHasRole(tt.Results(), role)
	case *types.Tuple:
		for v := range tt.Variables() {
			if f.typeHasRole(v.Type(), role) {
				return true
			}
		}
		return false
	case *types.Named:
		obj := tt.Obj()
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		match, ok := f.runner.cfg.ComponentForPath(f.runner.relImportPath(obj.Pkg().Path()))
		if ok && match.Role == role {
			return true
		}
		return f.typeHasRole(tt.Underlying(), role)
	default:
		return false
	}
}

func exprFromNode(n ast.Node) ast.Expr {
	expr, _ := n.(ast.Expr)
	return expr
}

func signalKinds(signals []businessLogicSignal) []string {
	out := make([]string, 0, len(signals))
	for _, signal := range signals {
		out = append(out, signal.kind)
	}
	return out
}

func containsWord(name, word string) bool {
	if word == "" || len(word) > len(name) {
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
