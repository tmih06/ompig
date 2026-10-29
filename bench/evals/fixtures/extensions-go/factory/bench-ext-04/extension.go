// SPDX-License-Identifier: MIT

// Package benchext04 is bench-ext-04: the conventional factory form (packed subprocess cell, or fused into a Ompiglet Binary).
package benchext04

import (
	sdk "github.com/tmih06/ompig/extensions/sdk"

	"example.com/benchext10/benchext"
)

func Extension() *sdk.Extension { return benchext.New("04") }
