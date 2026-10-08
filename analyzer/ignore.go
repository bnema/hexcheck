package analyzer

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/bnema/hexcheck/config"
	"golang.org/x/tools/go/analysis"
)

// DirectiveCategory is the diagnostic category used for malformed
// //hexcheck:ignore directives.
const DirectiveCategory = "invalid-ignore-directive"

const ignoreDirective = "//hexcheck:ignore"

type ignoreKey struct {
	file string
	line int
	rule string
}

// ignoreIndex records which rules are suppressed on which lines.
type ignoreIndex map[ignoreKey]bool

// collectIgnores indexes `//hexcheck:ignore rule[,rule] reason` directives.
// A directive suppresses matching diagnostics on its own line and on the
// following line. A reason is required, like config allow entries.
func (r runner) collectIgnores(file *ast.File) {
	known := config.DefaultRuleSeverities()
	for _, group := range file.Comments {
		for _, comment := range group.List {
			rest, ok := strings.CutPrefix(comment.Text, ignoreDirective)
			if !ok || (rest != "" && rest[0] != ' ' && rest[0] != '\t') {
				continue
			}
			fields := strings.Fields(rest)
			if len(fields) < 2 {
				r.reportDirective(comment.Pos(), "hexcheck:ignore needs a rule list and a reason")
				continue
			}
			pos := r.pass.Fset.Position(comment.Pos())
			for rule := range strings.SplitSeq(fields[0], ",") {
				if _, ok := known[rule]; !ok {
					r.reportDirective(comment.Pos(), "hexcheck:ignore names unknown rule "+rule)
					continue
				}
				r.ignores[ignoreKey{file: pos.Filename, line: pos.Line, rule: rule}] = true
				r.ignores[ignoreKey{file: pos.Filename, line: pos.Line + 1, rule: rule}] = true
			}
		}
	}
}

func (r runner) isIgnoredInline(pos token.Pos, rule string) bool {
	if r.ignores == nil {
		return false
	}
	p := r.pass.Fset.Position(pos)
	return r.ignores[ignoreKey{file: p.Filename, line: p.Line, rule: rule}]
}

func (r runner) reportDirective(pos token.Pos, msg string) {
	r.pass.Report(analysis.Diagnostic{Pos: pos, Category: DirectiveCategory, Message: "[error] " + DirectiveCategory + ": " + msg})
}
