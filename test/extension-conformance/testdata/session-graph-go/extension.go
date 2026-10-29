package sessiongraph

import (
	sdk "github.com/tmih06/ompig/extensions/sdk"
	"github.com/tmih06/ompig/test/extension-conformance/testfixture/sessiongraph"
)

func Extension() *sdk.Extension { return sessiongraph.Extension() }
