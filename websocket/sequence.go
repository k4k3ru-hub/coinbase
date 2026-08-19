package websocket

import "sync"

// SequenceState describes the relationship between an observed and prior sequence.
type SequenceState string

const (
	SequenceFirst      SequenceState = "first"
	SequenceNext       SequenceState = "next"
	SequenceDuplicate  SequenceState = "duplicate"
	SequenceGap        SequenceState = "gap"
	SequenceOutOfOrder SequenceState = "out_of_order"
)

// SequenceResult reports a sequence observation without assuming recovery is safe.
type SequenceResult struct {
	State             SequenceState
	Previous, Current uint64
}

// SequenceTracker validates monotonic sequences independently for each product.
type SequenceTracker struct {
	mu   sync.Mutex
	last map[string]uint64
}

// NewSequenceTracker creates an empty sequence tracker.
//
// Version:
//   - 2026-08-19: Added.
func NewSequenceTracker() *SequenceTracker { return &SequenceTracker{last: map[string]uint64{}} }

// Observe classifies and records a product sequence. Duplicates and out-of-order values do not move the cursor.
//
// Version:
//   - 2026-08-19: Added.
func (t *SequenceTracker) Observe(productID string, current uint64) SequenceResult {
	t.mu.Lock()
	defer t.mu.Unlock()
	previous, ok := t.last[productID]
	if !ok {
		t.last[productID] = current
		return SequenceResult{State: SequenceFirst, Current: current}
	}
	result := SequenceResult{Previous: previous, Current: current}
	switch {
	case current == previous:
		result.State = SequenceDuplicate
	case current < previous:
		result.State = SequenceOutOfOrder
	case current == previous+1:
		result.State = SequenceNext
		t.last[productID] = current
	default:
		result.State = SequenceGap
		t.last[productID] = current
	}
	return result
}

// Reset removes all sequence history, as required after reconnect.
//
// Version:
//   - 2026-08-19: Added.
func (t *SequenceTracker) Reset() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.last = map[string]uint64{}
}
