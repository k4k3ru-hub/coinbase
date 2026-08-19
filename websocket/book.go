package websocket

import (
	"fmt"
	"sync"
)

// Book is an immutable copy of one L2 product book.
type Book struct {
	ProductID    string
	Bids         map[string]string
	Asks         map[string]string
	Synchronized bool
	Generation   uint64
}

// BookManager builds L2 state from snapshots and absolute updates.
type BookManager struct {
	mu         sync.RWMutex
	books      map[string]*Book
	generation uint64
}

// NewBookManager creates an empty L2 manager.
//
// Version:
//   - 2026-08-19: Added.
func NewBookManager() *BookManager { return &BookManager{books: map[string]*Book{}, generation: 1} }

// Reset invalidates all books for reconnect, gap, or slow-consumer recovery.
//
// Version:
//   - 2026-08-19: Added.
func (m *BookManager) Reset() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generation++
	m.books = map[string]*Book{}
}

// Apply applies a snapshot or L2 update. Updates before a snapshot are rejected.
//
// Version:
//   - 2026-08-19: Added.
func (m *BookManager) Apply(event Event) error {
	if m == nil {
		return fmt.Errorf("failed to apply coinbase exchange L2 event: book_manager=null")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	switch v := event.Value.(type) {
	case *Level2Snapshot:
		b := &Book{ProductID: v.ProductID, Bids: map[string]string{}, Asks: map[string]string{}, Synchronized: true, Generation: m.generation}
		if err := loadLevels(b.Bids, v.Bids); err != nil {
			return fmt.Errorf("failed to apply coinbase exchange L2 snapshot: %w", err)
		}
		if err := loadLevels(b.Asks, v.Asks); err != nil {
			return fmt.Errorf("failed to apply coinbase exchange L2 snapshot: %w", err)
		}
		m.books[v.ProductID] = b
	case *Level2Update:
		b := m.books[v.ProductID]
		if b == nil || !b.Synchronized {
			return fmt.Errorf("failed to apply coinbase exchange L2 update: book=unsynchronized product_id=%q", v.ProductID)
		}
		for i, row := range v.Changes {
			if len(row) != 3 {
				return fmt.Errorf("failed to apply coinbase exchange L2 update: change_tuple=invalid index=%d actual_length=%d", i, len(row))
			}
			var levels map[string]string
			switch row[0] {
			case "buy":
				levels = b.Bids
			case "sell":
				levels = b.Asks
			default:
				return fmt.Errorf("failed to apply coinbase exchange L2 update: side=invalid")
			}
			if isZero(row[2]) {
				delete(levels, row[1])
			} else {
				levels[row[1]] = row[2]
			}
		}
	}
	return nil
}

// Snapshot returns a copy only when the current generation is synchronized.
//
// Version:
//   - 2026-08-19: Added.
func (m *BookManager) Snapshot(productID string) (Book, bool) {
	if m == nil {
		return Book{}, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	b := m.books[productID]
	if b == nil || !b.Synchronized {
		return Book{}, false
	}
	out := Book{ProductID: b.ProductID, Bids: copyMap(b.Bids), Asks: copyMap(b.Asks), Synchronized: true, Generation: b.Generation}
	return out, true
}
func loadLevels(dst map[string]string, rows [][]string) error {
	for i, row := range rows {
		if len(row) != 2 {
			return fmt.Errorf("level_tuple=invalid index=%d actual_length=%d", i, len(row))
		}
		dst[row[0]] = row[1]
	}
	return nil
}
func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func isZero(v string) bool {
	for _, r := range v {
		if r != '0' && r != '.' {
			return false
		}
	}
	return true
}
