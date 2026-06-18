package diff

import (
	"path/filepath"
	"strings"
)

func isProtected(relPath string) bool {
	if strings.EqualFold(filepath.Base(relPath), ".gitignore") {
		return true
	}
	for _, part := range strings.Split(filepath.ToSlash(relPath), "/") {
		if strings.EqualFold(part, ".git") {
			return true
		}
	}
	return false
}
