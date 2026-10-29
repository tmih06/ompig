package extension

import (
	"context"

	"github.com/tmih06/ompig/internal/childwait"
)

// WithExecDrainHooks returns ctx carrying hooks for the ExecCommand call that
// receives it: a test decides when the post-exit grace elapses and sees what
// the call's readers have consumed.
func WithExecDrainHooks(ctx context.Context, hooks *childwait.Hooks) context.Context {
	return context.WithValue(ctx, execDrainHooksKey{}, hooks)
}
