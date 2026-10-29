package extensiontest

import "github.com/tmih06/ompig/coding/extension"

// NewMemBus returns a fresh in-process event bus, [extension.CreateEventBus].
func NewMemBus() extension.EventBusController { return extension.CreateEventBus() }
