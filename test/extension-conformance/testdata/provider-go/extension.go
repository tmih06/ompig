package providerproducer

import (
	sdk "github.com/tmih06/ompig/extensions/sdk"
	"github.com/tmih06/ompig/test/extension-conformance/testfixture"
)

func Extension() *sdk.Extension { return testfixture.ProviderProducer() }
