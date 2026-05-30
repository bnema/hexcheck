package main

import (
	"github.com/bnema/hexcheck/analyzer"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(analyzer.New(analyzer.Options{}))
}
