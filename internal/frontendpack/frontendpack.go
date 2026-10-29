// Package frontendpack holds the frontend member fused into a Ompiglet Binary
// (D91). The build generates registry_generated.go, which registers the
// member's Frontend factory. Stock Ompig compiles this package with no member,
// so Selected returns nil and the interactive mode paints with its ANSI
// renderer.
package frontendpack

import "github.com/tmih06/ompig/extensions/sdk/frontend"

var selected func() frontend.Frontend

// register records the member's factory. A Ompiglet has at most one frontend.
func register(factory func() frontend.Frontend) { selected = factory }

// Selected constructs the fused frontend, or returns nil when none is fused.
func Selected() frontend.Frontend {
	if selected == nil {
		return nil
	}
	return selected()
}
