package extensionconformance

import "github.com/tmih06/ompig/coding/extension/host/subprocess"

// The conformance hosts serve the editors an extension's view holds, as the ompig command's host does.
func init() { subprocess.EnableEditorViews() }
