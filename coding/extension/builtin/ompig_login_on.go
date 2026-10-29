//go:build !ompig_strip_ompig_login

package builtin

import (
	"github.com/tmih06/ompig/coding/piglogin"
	"github.com/tmih06/ompig/internal/pigstrip"
)

// ompigLoginEntries is the ompig-login registry row (the `/sprite` command), absent while ompig-login is stripped at runtime as in a
// build without it.
// ompig additive (D92): a runtime stripped ompig-login registers nothing, like the ompig_strip_ompig_login build.
func ompigLoginEntries() []Extension {
	if pigstrip.Has(pigstrip.ListExtensions, piglogin.Name) {
		return nil
	}
	return []Extension{{Name: piglogin.Name, Replaceable: true, Factory: piglogin.Extension}}
}
