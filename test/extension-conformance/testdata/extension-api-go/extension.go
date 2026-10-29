package extensionapi

import (
	sdk "github.com/tmih06/ompig/extensions/sdk"
	"github.com/tmih06/ompig/test/extension-conformance/testfixture/extensionapi"
)

func Extension() *sdk.Extension { return extensionapi.Extension() }
