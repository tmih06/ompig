package subprocess

import (
	"github.com/tmih06/ompig/coding/extension"
	"github.com/tmih06/ompig/tui"
)

// remoteEditorOfFactory is the editor in an extension's process that factory builds (extension.RemoteEditorFactory); nil for a nil factory.
func remoteEditorOfFactory(factory extension.EditorFactory) extension.RemoteEditor {
	if factory == nil {
		return nil
	}
	editor, _ := extension.RemoteEditorOf(factory(nil, tui.EditorTheme{}, nil))
	return editor
}
