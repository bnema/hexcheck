// Command hexcheck runs the hexcheck analyzer on Go packages.
//
//	hexcheck [flags] [packages]   check packages (default ./...)
//	hexcheck init [-force] [dir]  write a starter .hexcheck.yaml
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/bnema/hexcheck/analyzer"
	"github.com/bnema/hexcheck/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

// Exit codes.
const (
	exitOK       = 0
	exitFailure  = 1 // load, config, or analysis failure
	exitUsage    = 2
	exitFindings = 3 // diagnostics at or above -fail-on
)

// version is set at release time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "init" {
		return runInit(args[1:], stdout, stderr)
	}

	fs := flag.NewFlagSet("hexcheck", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "path to .hexcheck.yaml (default: discovered from -root upwards)")
	root := fs.String("root", "", "project root for config-relative paths")
	modulePath := fs.String("module", "", "Go module path (default: module in go.mod)")
	tests := fs.Bool("test", true, "also analyze test files")
	jsonOut := fs.Bool("json", false, "print diagnostics as JSON")
	failOn := fs.String("fail-on", "error", "lowest severity that makes the exit code non-zero: error or warn")
	showVersion := fs.Bool("version", false, "print the version and exit")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: hexcheck [flags] [packages]\n       hexcheck init [-force] [dir]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if *showVersion {
		fmt.Fprintln(stdout, "hexcheck", version)
		return exitOK
	}
	if *failOn != string(config.SeverityError) && *failOn != string(config.SeverityWarn) {
		fmt.Fprintf(stderr, "hexcheck: -fail-on must be error or warn, got %q\n", *failOn)
		return exitUsage
	}
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	a := analyzer.New(analyzer.Options{ConfigPath: *configPath, Root: *root, ModulePath: *modulePath})
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadSyntax | packages.NeedModule, Tests: *tests, Dir: *root}, patterns...)
	if err != nil {
		fmt.Fprintf(stderr, "hexcheck: %v\n", err)
		return exitFailure
	}
	if packages.PrintErrors(pkgs) > 0 {
		return exitFailure
	}
	graph, err := checker.Analyze([]*analysis.Analyzer{a}, pkgs, nil)
	if err != nil {
		fmt.Fprintf(stderr, "hexcheck: %v\n", err)
		return exitFailure
	}

	if *jsonOut {
		if err := graph.PrintJSON(stdout); err != nil {
			return exitFailure
		}
	} else if err := graph.PrintText(stderr, -1); err != nil {
		return exitFailure
	}
	return exitCode(graph, config.Severity(*failOn))
}

// exitCode maps diagnostics to an exit code using each rule's configured
// severity, so warn-level rules do not fail CI unless -fail-on=warn.
func exitCode(graph *checker.Graph, failOn config.Severity) int {
	code := exitOK
	for act := range graph.All() {
		if act.Err != nil {
			return exitFailure
		}
		if !act.IsRoot {
			continue
		}
		result, _ := act.Result.(*analyzer.Result)
		for _, diag := range act.Diagnostics {
			severity := config.SeverityError
			if diag.Category != analyzer.DirectiveCategory {
				severity = result.Severity(diag.Category)
			}
			if severity == config.SeverityError || (failOn == config.SeverityWarn && severity == config.SeverityWarn) {
				code = exitFindings
			}
		}
	}
	return code
}
