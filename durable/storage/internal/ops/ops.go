// Package ops replays stored Chord operation batches for the storage backends.
package ops

import (
	"slices"

	"github.com/tmih06/ompig/chord/delta"
	"github.com/tmih06/ompig/durable"
)

// ApplyBatches replays decoded operation batches over base without mutating it and returns the final value only.
func ApplyBatches(base durable.JsonValue, batches [][]durable.Op) (durable.JsonValue, error) {
	return delta.ApplyImmutableBatchesSeq(base, slices.Values(batches))
}
