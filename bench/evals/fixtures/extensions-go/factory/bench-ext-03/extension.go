// SPDX-License-Identifier: MIT

// Package benchext03 is bench-ext-03: the conventional factory form (packed subprocess cell, or fused into a Ompiglet Binary).
package benchext03

import (
	sdk "github.com/tmih06/ompig/extensions/sdk"

	"example.com/benchext10/benchext"
)

func Extension() *sdk.Extension { return benchext.New("03") }
