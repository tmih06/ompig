//go:build !windows

package codingagent

import "github.com/tmih06/ompig/internal/realpath"

func evalCanonicalPath(path string) (string, error) {
	return realpath.EvalSymlinks(path)
}
