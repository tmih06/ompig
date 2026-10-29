//go:build ompig_strip_docs

package prompts

import "github.com/tmih06/ompig/internal/pigstrip"

// ompig additive (D92): this Ompiglet Binary compiled out the Ompig documentation bundle, so the prompt has no docs section.
func init() { pigstrip.Strip(pigstrip.ListFeatures, pigstrip.Docs) }

// docsSections is empty: this build writes no docs for the model to read.
func docsSections(string, string, string) []section { return nil }
