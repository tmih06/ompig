package providerobject

import (
	sdk "github.com/tmih06/ompig/extensions/sdk"
	"github.com/tmih06/ompig/test/extension-conformance/testfixture/providerobject"
)

func Extension() *sdk.Extension { return providerobject.Extension() }
