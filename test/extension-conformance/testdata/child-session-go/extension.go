package childsession

import (
	sdk "github.com/tmih06/ompig/extensions/sdk"
	"github.com/tmih06/ompig/test/extension-conformance/testfixture/childsession"
)

func Extension() *sdk.Extension { return childsession.Extension() }
