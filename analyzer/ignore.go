package analyzer

import (
	"go/ast"
	"go/token"
	"sort"
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
type ignoreIndex map[ignoreKey][]*ignoreUsage

type ignoreUsage struct {
	pos      token.Pos
	rule     string
	used     bool
	filePath string
}

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
				usage := &ignoreUsage{pos: comment.Pos(), rule: rule, filePath: r.filePath(file)}
				for _, line := range []int{pos.Line, pos.Line + 1} {
					key := ignoreKey{file: pos.Filename, line: line, rule: rule}
					r.ignores[key] = append(r.ignores[key], usage)
				}
			}
		}
	}
}

func (r runner) isIgnoredInline(pos token.Pos, rule string) bool {
	if r.ignores == nil {
		return false
	}
	p := r.pass.Fset.Position(pos)
	usages := r.ignores[ignoreKey{file: p.Filename, line: p.Line, rule: rule}]
	for _, usage := range usages {
		usage.used = true
	}
	return len(usages) > 0
}

func (r runner) reportUnusedIgnores() {
	seen := map[*ignoreUsage]bool{}
	var unused []*ignoreUsage
	for _, usages := range r.ignores {
		for _, usage := range usages {
			if !seen[usage] && !usage.used {
				unused = append(unused, usage)
			}
			seen[usage] = true
		}
	}
	sort.Slice(unused, func(i, j int) bool {
		if unused[i].pos != unused[j].pos {
			return unused[i].pos < unused[j].pos
		}
		return unused[i].rule < unused[j].rule
	})
	for _, usage := range unused {
		r.report(usage.pos, "unused-ignore-directive", usage.filePath, "hexcheck:ignore %s suppresses no diagnostics", usage.rule)
	}
}

func (r runner) reportDirective(pos token.Pos, msg string) {
	r.pass.Report(analysis.Diagnostic{Pos: pos, Category: DirectiveCategory, Message: "[error] " + DirectiveCategory + ": " + msg})
}
