//go:build ompig_strip_ompig_login

package builtin

import "github.com/tmih06/ompig/internal/pigstrip"

// ompig additive (D92): a build without ompig-login records it as stripped, so `builtin:ompig-login` is filtered out silently. The
// piglogin package stays linked because internal/codingagent draws its sprite; only the `/sprite` extension leaves.
func init() { pigstrip.Strip(pigstrip.ListExtensions, "ompig-login") }

func ompigLoginEntries() []Extension { return nil }
