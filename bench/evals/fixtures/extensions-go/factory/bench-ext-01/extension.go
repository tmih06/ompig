// SPDX-License-Identifier: MIT

// Package benchext01 is bench-ext-01: the conventional factory form (packed subprocess cell, or fused into a Ompiglet Binary).
package benchext01

import (
	sdk "github.com/tmih06/ompig/extensions/sdk"

	"example.com/benchext10/benchext"
)

func Extension() *sdk.Extension { return benchext.New("01") }
