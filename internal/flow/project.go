package flow

import (
	"os"
	"path/filepath"
)

// ProjectOf names the project a folder belongs to: the folder that holds the
// nearest .git above it, or the folder itself when there is none.
func ProjectOf(dir string) string {
	d := filepath.Clean(dir)
	for cur := d; ; {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return filepath.Base(cur)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return filepath.Base(d)
}
