package glob

import (
	"path"
	"strings"
)

func Normalize(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "./")
	return strings.Trim(p, "/")
}

func Match(pattern, name string) bool {
	pattern = Normalize(pattern)
	name = Normalize(name)
	if pattern == "" {
		return name == ""
	}
	return matchParts(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

func matchParts(pattern, name []string) bool {
	if len(pattern) == 0 {
		return len(name) == 0
	}
	if pattern[0] == "**" {
		if len(pattern) == 1 {
			return true
		}
		for i := 0; i <= len(name); i++ {
			if matchParts(pattern[1:], name[i:]) {
				return true
			}
		}
		return false
	}
	if len(name) == 0 {
		return false
	}
	ok, err := path.Match(pattern[0], name[0])
	if err != nil || !ok {
		return false
	}
	return matchParts(pattern[1:], name[1:])
}

func Specificity(pattern string) int {
	pattern = Normalize(pattern)
	cut := strings.IndexAny(pattern, "*?[{")
	if cut == -1 {
		return len(pattern)
	}
	return cut
}
