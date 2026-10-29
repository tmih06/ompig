package coding

import "github.com/tmih06/ompig/tui"

// ModelRuntime is the runtime a tui.ModelSelectorRuntime reads (model-selector.ts takes a ModelRuntime; package tui cannot import this one).
var _ tui.ModelSelectorRuntime = (*ModelRuntime)(nil)
