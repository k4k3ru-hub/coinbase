package websocket

import (
	"fmt"
	"math/big"
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

// BBO is the best bid and offer for one synchronized product book.
type BBO struct {
	ProductID  string
	BidPrice   string
	BidSize    string
	AskPrice   string
	AskSize    string
	Generation uint64
}

type managedBook struct {
	book         *Book
	bestBidPrice string
	bestAskPrice string
}

// BookManager builds L2 state from snapshots and absolute updates.
type BookManager struct {
	mu         sync.RWMutex
	books      map[string]*managedBook
	generation uint64
}

// NewBookManager creates an empty L2 manager.
//
// Version:
//   - 2026-08-19: Added.
func NewBookManager() *BookManager {
	return &BookManager{books: map[string]*managedBook{}, generation: 1}
}

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
	m.books = map[string]*managedBook{}
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
		bestBidPrice, err := bestPrice(b.Bids, true)
		if err != nil {
			return fmt.Errorf("failed to apply coinbase exchange L2 snapshot: invalid bid price: %w", err)
		}
		bestAskPrice, err := bestPrice(b.Asks, false)
		if err != nil {
			return fmt.Errorf("failed to apply coinbase exchange L2 snapshot: invalid ask price: %w", err)
		}
		m.books[v.ProductID] = &managedBook{book: b, bestBidPrice: bestBidPrice, bestAskPrice: bestAskPrice}
	case *Level2Update:
		managed := m.books[v.ProductID]
		if managed == nil || managed.book == nil || !managed.book.Synchronized {
			return fmt.Errorf("failed to apply coinbase exchange L2 update: book=unsynchronized product_id=%q", v.ProductID)
		}
		b := managed.book
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
			highest := row[0] == "buy"
			current := managed.bestAskPrice
			if highest {
				current = managed.bestBidPrice
			}
			best, err := updateBestPrice(levels, row[1], row[2], current, highest)
			if err != nil {
				return fmt.Errorf("failed to apply coinbase exchange L2 update: invalid price: %w", err)
			}
			if highest {
				managed.bestBidPrice = best
			} else {
				managed.bestAskPrice = best
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
	managed := m.books[productID]
	if managed == nil || managed.book == nil || !managed.book.Synchronized {
		return Book{}, false
	}
	b := managed.book
	out := Book{ProductID: b.ProductID, Bids: copyMap(b.Bids), Asks: copyMap(b.Asks), Synchronized: true, Generation: b.Generation}
	return out, true
}

// BestBidOffer returns the best bid and offer without copying the full book.
//
// Parameters:
//   - productID: Coinbase Exchange product identifier.
//
// Returns:
//   - Best bid and offer when both sides are available and synchronized.
//   - True when a complete BBO is available.
//
// Version:
//   - 2026-08-24: Added.
func (m *BookManager) BestBidOffer(productID string) (BBO, bool) {
	if m == nil {
		return BBO{}, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	managed := m.books[productID]
	if managed == nil || managed.book == nil || !managed.book.Synchronized || managed.bestBidPrice == "" || managed.bestAskPrice == "" {
		return BBO{}, false
	}
	return BBO{
		ProductID:  productID,
		BidPrice:   managed.bestBidPrice,
		BidSize:    managed.book.Bids[managed.bestBidPrice],
		AskPrice:   managed.bestAskPrice,
		AskSize:    managed.book.Asks[managed.bestAskPrice],
		Generation: managed.book.Generation,
	}, true
}

func bestPrice(levels map[string]string, highest bool) (string, error) {
	best := ""
	var bestValue *big.Rat
	for price := range levels {
		value, ok := new(big.Rat).SetString(price)
		if !ok || value.Sign() <= 0 {
			return "", fmt.Errorf("price=invalid")
		}
		if bestValue == nil || highest && value.Cmp(bestValue) > 0 || !highest && value.Cmp(bestValue) < 0 {
			best = price
			bestValue = value
		}
	}
	return best, nil
}

func updateBestPrice(levels map[string]string, price string, quantity string, current string, highest bool) (string, error) {
	value, ok := new(big.Rat).SetString(price)
	if !ok || value.Sign() <= 0 {
		return "", fmt.Errorf("price=invalid")
	}
	if isZero(quantity) {
		delete(levels, price)
		if price == current {
			return bestPrice(levels, highest)
		}
		return current, nil
	}
	levels[price] = quantity
	if current == "" {
		return price, nil
	}
	currentValue, ok := new(big.Rat).SetString(current)
	if !ok || currentValue.Sign() <= 0 {
		return "", fmt.Errorf("current_price=invalid")
	}
	if highest && value.Cmp(currentValue) > 0 || !highest && value.Cmp(currentValue) < 0 {
		return price, nil
	}
	return current, nil
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
