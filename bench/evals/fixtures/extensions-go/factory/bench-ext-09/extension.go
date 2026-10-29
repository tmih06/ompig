// SPDX-License-Identifier: MIT

// Package benchext09 is bench-ext-09: the conventional factory form (packed subprocess cell, or fused into a Ompiglet Binary).
package benchext09

import (
	sdk "github.com/tmih06/ompig/extensions/sdk"

	"example.com/benchext10/benchext"
)

func Extension() *sdk.Extension { return benchext.New("09") }
