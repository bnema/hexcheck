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

func (r runner) checkBusinessLogic(file *ast.File, filePath string, current config.Match) {
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
		r.isSuspiciousBusinessFunction(fn, file, filePath, current.Role)
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
	// Reject oversized bodies before type queries or secondary traversals.
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		facts.nodeCount++
		return facts.nodeCount <= *r.cfg.Heuristics.BusinessLogicMaxFunctionNodes
	})
	if facts.nodeCount > *r.cfg.Heuristics.BusinessLogicMaxFunctionNodes {
		return facts
	}
	facts.collectLocals()
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
	domainDecision        bool
	zeroDecision          bool
	guardedDomainError    bool
	domainMethodCall      bool
	domainConstructorCall bool
	domainErrorReturn     bool
	policyConstant        bool
	usecaseCalls          map[string]bool
	imports               map[string]bool
	usesTechnicalPackage  bool
	constructedLocals     map[types.Object]bool
}

func (f *businessFunctionFacts) inspect(n ast.Node) bool {
	if n == nil {
		return true
	}

	if f.exprUsesRole(exprFromNode(n), config.RoleCore) || f.exprUsesRole(exprFromNode(n), config.RoleUsecase) {
		f.domainTypeUse = true
	}

	switch node := n.(type) {
	case *ast.IfStmt:
		f.branchCount++
		f.inspectDecision(node.Cond)
		f.inspectMembershipInit(node)
		f.inspectPolicyError(node)
	case *ast.SwitchStmt:
		f.branchCount++
		f.inspectDecision(node.Tag)
	case *ast.CaseClause:
		for _, expr := range node.List {
			f.inspectDecision(expr)
		}
	case *ast.TypeSwitchStmt, *ast.ForStmt, *ast.RangeStmt:
		f.branchCount++
	case *ast.AssignStmt:
		for _, lhs := range node.Lhs {
			if f.selectorReceiverUsesRole(lhs, config.RoleCore, config.RoleUsecase) && !f.isConstructedLocal(lhs) {
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
			if f.exprUsesRole(result, config.RoleCore) || f.exprUsesRole(result, config.RoleUsecase) || f.isDomainError(result) {
				f.domainErrorReturn = true
			}
		}
	}
	return true
}

// inspectDecision distinguishes domain decisions from merely transporting domain
// values. Formatting calls do not become policy evidence in a comparison.
func (f *businessFunctionFacts) inspectDecision(expr ast.Expr) {
	if expr == nil {
		return
	}
	ast.Inspect(expr, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IndexExpr:
			// A domain-derived key deciding membership is policy, unlike
			// formatting a key for a port call.
			ast.Inspect(node.Index, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && f.exprUsesRole(sel.X, config.RoleCore) {
						f.domainDecision = true
					}
				}
				return true
			})
		case *ast.CallExpr:
			if fn, ok := f.calleeObject(node.Fun).(*types.Func); ok && f.coreObject(fn) {
				sig := fn.Type().(*types.Signature)
				if sig.Recv() == nil && sig.Results().Len() == 1 && types.Identical(sig.Results().At(0).Type(), types.Typ[types.Bool]) {
					f.domainDecision = true
				}
			}
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok && f.exprUsesRole(sel.X, config.RoleCore) {
				switch sel.Sel.Name {
				case "String", "Error":
				case "IsZero":
					f.zeroDecision = true
				default:
					f.domainDecision = true
				}
			}
			return false
		case *ast.BinaryExpr:
			switch node.Op {
			case token.LSS, token.LEQ, token.GTR, token.GEQ:
				if f.numericDomainOperand(node.X) || f.numericDomainOperand(node.Y) {
					f.domainDecision = true
				}
			}
		}
		return true
	})
}

func (f *businessFunctionFacts) isDomainError(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || !strings.HasPrefix(sel.Sel.Name, "Err") {
		return false
	}
	obj := f.runner.pass.TypesInfo.ObjectOf(sel.Sel)
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	match, ok := f.runner.cfg.ComponentForPath(f.runner.relImportPath(obj.Pkg().Path()))
	return ok && match.Role == config.RoleCore
}

func (f *businessFunctionFacts) calleeObject(expr ast.Expr) types.Object {
	switch expr := expr.(type) {
	case *ast.Ident:
		return f.runner.pass.TypesInfo.ObjectOf(expr)
	case *ast.SelectorExpr:
		return f.runner.pass.TypesInfo.ObjectOf(expr.Sel)
	}
	return nil
}

func (f *businessFunctionFacts) coreObject(obj types.Object) bool {
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	match, ok := f.runner.cfg.ComponentForPath(f.runner.relImportPath(obj.Pkg().Path()))
	return ok && match.Role == config.RoleCore
}

func (f *businessFunctionFacts) numericDomainOperand(expr ast.Expr) bool {
	t := f.runner.pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	basic, ok := t.Underlying().(*types.Basic)
	if !ok || basic.Info()&(types.IsInteger|types.IsFloat) == 0 {
		return false
	}
	switch expr := expr.(type) {
	case *ast.ParenExpr:
		return f.numericDomainOperand(expr.X)
	case *ast.SelectorExpr:
		return f.exprUsesRole(expr.X, config.RoleCore) || f.coreObject(f.runner.pass.TypesInfo.ObjectOf(expr.Sel))
	case *ast.Ident:
		return f.coreObject(f.runner.pass.TypesInfo.ObjectOf(expr))
	}
	return false
}

func (f *businessFunctionFacts) inspectMembershipInit(stmt *ast.IfStmt) {
	assign, ok := stmt.Init.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
		return
	}
	index, ok := assign.Rhs[0].(*ast.IndexExpr)
	if !ok {
		return
	}
	t := f.runner.pass.TypesInfo.TypeOf(index.X)
	if t == nil {
		return
	}
	if _, ok := t.Underlying().(*types.Map); !ok {
		return
	}
	id, ok := assign.Lhs[1].(*ast.Ident)
	if !ok || id.Name == "_" {
		return
	}
	obj := f.runner.pass.TypesInfo.ObjectOf(id)
	// Only follow the comma-ok boolean when it participates in the condition.
	ast.Inspect(stmt.Cond, func(n ast.Node) bool {
		if use, ok := n.(*ast.Ident); ok && f.runner.pass.TypesInfo.ObjectOf(use) == obj {
			f.inspectDecision(index)
		}
		return true
	})
}

// A guarded domain error is policy evidence unless it is a conventional
// storage/validation sentinel. Those are routinely translated by adapters.
func (f *businessFunctionFacts) inspectPolicyError(stmt *ast.IfStmt) {
	for _, s := range stmt.Body.List {
		ret, ok := s.(*ast.ReturnStmt)
		if !ok {
			continue
		}
		for _, result := range ret.Results {
			sel, ok := result.(*ast.SelectorExpr)
			if !ok || !strings.HasPrefix(sel.Sel.Name, "Err") {
				continue
			}
			switch sel.Sel.Name {
			case "ErrNotFound", "ErrConflict", "ErrInvalid", "ErrAlreadyExists":
				continue
			}
			if f.isDomainError(result) {
				f.domainDecision = true
				f.guardedDomainError = true
			}
		}
	}
}

// isConstructedLocal excludes field-by-field decoding into a fresh local
// value. Values obtained from calls (including ports) remain mutation evidence.
func (f *businessFunctionFacts) isConstructedLocal(expr ast.Expr) bool {
	for {
		sel, ok := expr.(*ast.SelectorExpr)
		if !ok {
			break
		}
		expr = sel.X
	}
	id, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return f.constructedLocals[f.runner.pass.TypesInfo.ObjectOf(id)]
}

func (f *businessFunctionFacts) collectLocals() {
	f.constructedLocals = map[types.Object]bool{}
	reassigned := map[types.Object]bool{}
	info := f.runner.pass.TypesInfo
	ast.Inspect(f.fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ValueSpec:
			for i, name := range node.Names {
				if obj := info.Defs[name]; obj != nil {
					f.constructedLocals[obj] = len(node.Values) == 0 || (i < len(node.Values) && isFreshValue(node.Values[i]))
				}
			}
		case *ast.AssignStmt:
			for i, lhs := range node.Lhs {
				if name, ok := lhs.(*ast.Ident); ok {
					obj := info.ObjectOf(name)
					if obj == nil {
						continue
					}
					if info.Defs[name] == obj {
						f.constructedLocals[obj] = i < len(node.Rhs) && isFreshValue(node.Rhs[i])
					} else {
						reassigned[obj] = true
					}
				}
			}
		}
		return true
	})
	for obj := range reassigned {
		delete(f.constructedLocals, obj)
	}
}

func isFreshValue(expr ast.Expr) bool {
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expr = unary.X
	}
	_, ok := expr.(*ast.CompositeLit)
	return ok
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

	if f.zeroDecision && f.domainMutation {
		f.domainDecision = true
	}
	if f.domainMutation {
		weak = append(weak, businessLogicSignal{kind: "domain mutation"})
	}
	if f.domainMethodCall {
		weak = append(weak, businessLogicSignal{kind: "domain method call"})
	}
	if f.guardedDomainError && f.branchCount >= 2 {
		weak = append(weak, businessLogicSignal{kind: "guarded domain error"})
	}
	if f.domainDecision && f.branchCount >= 2 {
		strong = append(strong, businessLogicSignal{kind: "domain-driven control flow", strong: true})
	}
	if f.keyword && len(f.usecaseCalls) >= 2 {
		strong = append(strong, businessLogicSignal{kind: "business keyword with multiple port/usecase collaborators", strong: true})
	}
	if f.keyword && f.branchCount >= 3 && f.policyConstant && policyKeyword(f.fn.Name.Name) {
		strong = append(strong, businessLogicSignal{kind: "business keyword with policy constants and branching", strong: true})
	}
	if f.keyword && (f.domainDecision || (f.domainMutation && containsWord(f.fn.Name.Name, "Validate")) || (f.domainTypeUse && (containsWord(f.fn.Name.Name, "Compute") || containsWord(f.fn.Name.Name, "Calculate") || containsWord(f.fn.Name.Name, "Score")))) {
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
