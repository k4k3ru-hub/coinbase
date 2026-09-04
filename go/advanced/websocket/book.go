package websocket

import (
	"fmt"
	"math/big"
	"sync"
)

// BBO is the best bid and offer for one synchronized Advanced Trade product.
type BBO struct {
	ProductID  string
	BidPrice   string
	BidSize    string
	AskPrice   string
	AskSize    string
	Generation uint64
}

// Book is an immutable copy of one synchronized Advanced Trade product book.
type Book struct {
	ProductID  string
	Bids       map[string]string
	Asks       map[string]string
	Generation uint64
}
type book struct {
	bids       map[string]string
	asks       map[string]string
	bestBid    string
	bestAsk    string
	generation uint64
}

// BookManager maintains books from absolute level2 snapshots and updates.
type BookManager struct {
	mu         sync.RWMutex
	books      map[string]*book
	generation uint64
}

// NewBookManager creates an empty Advanced Trade L2 manager.
//
// Version:
//   - 2026-08-24: Added.
func NewBookManager() *BookManager { return &BookManager{books: map[string]*book{}, generation: 1} }

// Reset invalidates every book after reconnect, sequence gap, or overflow.
//
// Version:
//   - 2026-08-24: Added.
func (m *BookManager) Reset() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generation++
	m.books = map[string]*book{}
}

// Apply applies all L2 batches in an envelope.
//
// Version:
//   - 2026-08-24: Added.
func (m *BookManager) Apply(event Event) error {
	if m == nil {
		return fmt.Errorf("failed to apply coinbase advanced L2 event: book_manager=null")
	}
	envelope, ok := event.Value.(*Level2Envelope)
	if !ok {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, batch := range envelope.Events {
		if batch.ProductID == "" {
			return fmt.Errorf("failed to apply coinbase advanced L2 event: product_id=empty")
		}
		current := m.books[batch.ProductID]
		switch batch.Type {
		case "snapshot":
			current = &book{bids: map[string]string{}, asks: map[string]string{}, generation: m.generation}
			m.books[batch.ProductID] = current
		case "update":
			if current == nil {
				return fmt.Errorf("failed to apply coinbase advanced L2 update: book=unsynchronized product_id=%q", batch.ProductID)
			}
		default:
			continue
		}
		for _, update := range batch.Updates {
			levels := current.bids
			highest := true
			switch update.Side {
			case "bid":
			case "offer", "ask":
				levels = current.asks
				highest = false
			default:
				return fmt.Errorf("failed to apply coinbase advanced L2 event: side=invalid product_id=%q", batch.ProductID)
			}
			best := current.bestBid
			if !highest {
				best = current.bestAsk
			}
			next, err := updateLevel(levels, update.PriceLevel, update.NewQuantity, best, highest)
			if err != nil {
				return fmt.Errorf("failed to apply coinbase advanced L2 event: %w: product_id=%q", err, batch.ProductID)
			}
			if highest {
				current.bestBid = next
			} else {
				current.bestAsk = next
			}
		}
	}
	return nil
}

// BestBidOffer returns a BBO only for a synchronized two-sided book.
//
// Version:
//   - 2026-08-24: Added.
func (m *BookManager) BestBidOffer(productID string) (BBO, bool) {
	if m == nil {
		return BBO{}, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	current := m.books[productID]
	if current == nil || current.bestBid == "" || current.bestAsk == "" {
		return BBO{}, false
	}
	return BBO{ProductID: productID, BidPrice: current.bestBid, BidSize: current.bids[current.bestBid], AskPrice: current.bestAsk, AskSize: current.asks[current.bestAsk], Generation: current.generation}, true
}

// Snapshot returns a detached copy of a synchronized order book.
//
// Parameters:
//   - productID: Advanced Trade product identifier.
//
// Returns:
//   - Detached order book.
//   - True when a synchronized book is available.
//
// Version:
//   - 2026-09-04: Added.
func (m *BookManager) Snapshot(productID string) (Book, bool) {
	if m == nil {
		return Book{}, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	current := m.books[productID]
	if current == nil || current.bestBid == "" || current.bestAsk == "" {
		return Book{}, false
	}
	return Book{
		ProductID: productID,
		Bids:      copyLevels(current.bids),
		Asks:      copyLevels(current.asks), Generation: current.generation,
	}, true
}

func copyLevels(levels map[string]string) map[string]string {
	result := make(map[string]string, len(levels))
	for price, quantity := range levels {
		result[price] = quantity
	}
	return result
}

func updateLevel(levels map[string]string, price, quantity, current string, highest bool) (string, error) {
	value, ok := new(big.Rat).SetString(price)
	if !ok || value.Sign() <= 0 {
		return "", fmt.Errorf("price=invalid")
	}
	if zero(quantity) {
		delete(levels, price)
		if price == current {
			return best(levels, highest)
		}
		return current, nil
	}
	levels[price] = quantity
	if current == "" {
		return price, nil
	}
	currentValue, ok := new(big.Rat).SetString(current)
	if !ok {
		return "", fmt.Errorf("current_price=invalid")
	}
	if highest && value.Cmp(currentValue) > 0 || !highest && value.Cmp(currentValue) < 0 {
		return price, nil
	}
	return current, nil
}
func best(levels map[string]string, highest bool) (string, error) {
	result := ""
	var resultValue *big.Rat
	for price := range levels {
		value, ok := new(big.Rat).SetString(price)
		if !ok || value.Sign() <= 0 {
			return "", fmt.Errorf("price=invalid")
		}
		if resultValue == nil || highest && value.Cmp(resultValue) > 0 || !highest && value.Cmp(resultValue) < 0 {
			result = price
			resultValue = value
		}
	}
	return result, nil
}
func zero(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r != '0' && r != '.' {
			return false
		}
	}
	return true
}
