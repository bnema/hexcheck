package analyzer

import (
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestBusinessLogicGoldenCases(t *testing.T) {
	result := analysistest.Run(t, analysistest.TestData(), New(Options{Config: testConfig(), ModulePath: "example.com/project"}), "example.com/project/internal/infrastructure/http")

	messagesByFile := map[string][]string{}
	for _, diagnostics := range result[0].Diagnostics {
		file := filepath.ToSlash(result[0].Pass.Fset.Position(diagnostics.Pos).Filename)
		messagesByFile[file] = append(messagesByFile[file], diagnostics.Message)
	}

	assertHasBusinessLogicDiagnostic(t, messagesByFile, "typed_policy.go")
	assertHasBusinessLogicDiagnostic(t, messagesByFile, "mixed_detection.go")
	assertNoBusinessLogicDiagnostic(t, messagesByFile, "probing.go")
}

func assertHasBusinessLogicDiagnostic(t *testing.T, messagesByFile map[string][]string, fileSuffix string) {
	t.Helper()
	for file, messages := range messagesByFile {
		if !strings.HasSuffix(file, fileSuffix) {
			continue
		}
		for _, message := range messages {
			if strings.HasPrefix(message, "suspicious-business-logic-in-adapter:") {
				return
			}
		}
	}
	t.Fatalf("expected business-logic diagnostic for %s", fileSuffix)
}

func assertNoBusinessLogicDiagnostic(t *testing.T, messagesByFile map[string][]string, fileSuffix string) {
	t.Helper()
	for file, messages := range messagesByFile {
		if !strings.HasSuffix(file, fileSuffix) {
			continue
		}
		for _, message := range messages {
			if strings.HasPrefix(message, "suspicious-business-logic-in-adapter:") {
				t.Fatalf("unexpected business-logic diagnostic for %s: %s", fileSuffix, message)
			}
		}
	}
}
