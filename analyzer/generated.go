package analyzer

import (
	"go/ast"
	"strings"
)

func isGeneratedFile(file *ast.File) bool {
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			text = strings.TrimSpace(strings.TrimPrefix(text, "/*"))
			if strings.HasPrefix(text, "Code generated ") && strings.Contains(text, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}
