package websocket

import (
	"fmt"
	"testing"
)

func TestDecodeAndBook(t *testing.T) {
	t.Parallel()
	m := NewBookManager()
	snapshot, err := DecodeEvent([]byte(`{"type":"snapshot","product_id":"BTC-USD","bids":[["1","2"]],"asks":[["3","4"]]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Apply(snapshot); err != nil {
		t.Fatal(err)
	}
	update, err := DecodeEvent([]byte(`{"type":"l2update","product_id":"BTC-USD","time":"x","changes":[["buy","1","0"],["sell","3","5"]]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Apply(update); err != nil {
		t.Fatal(err)
	}
	b, ok := m.Snapshot("BTC-USD")
	if !ok || len(b.Bids) != 0 || b.Asks["3"] != "5" {
		t.Fatalf("book: %#v", b)
	}
	m.Reset()
	if _, ok = m.Snapshot("BTC-USD"); ok {
		t.Fatal("reset book remained synchronized")
	}
	if err = m.Apply(update); err == nil {
		t.Fatal("update without snapshot accepted")
	}
}

func TestDecodeRouting(t *testing.T) {
	t.Parallel()
	cases := map[string]any{"heartbeat": &Heartbeat{}, "status": &Status{}, "ticker": &Ticker{}, "subscriptions": &Subscriptions{}, "error": &ErrorMessage{}, "match": &Match{}, "open": &FullEvent{}}
	for typ, want := range cases {
		e, err := DecodeEvent([]byte(`{"type":"` + typ + `"}`))
		if err != nil {
			t.Fatal(err)
		}
		if fmtType(e.Value) != fmtType(want) {
			t.Fatalf("%s: %T", typ, e.Value)
		}
	}
	e, err := DecodeEvent([]byte(`{"type":"future_message","x":1}`))
	if err != nil || e.Value != nil || len(e.Raw) == 0 {
		t.Fatalf("unknown: %#v %v", e, err)
	}
}
func fmtType(v any) string { return fmt.Sprintf("%T", v) }

func TestSequenceTracker(t *testing.T) {
	t.Parallel()
	s := NewSequenceTracker()
	states := []SequenceState{s.Observe("BTC-USD", 10).State, s.Observe("BTC-USD", 11).State, s.Observe("BTC-USD", 11).State, s.Observe("BTC-USD", 9).State, s.Observe("BTC-USD", 14).State}
	want := []SequenceState{SequenceFirst, SequenceNext, SequenceDuplicate, SequenceOutOfOrder, SequenceGap}
	for i := range want {
		if states[i] != want[i] {
			t.Fatalf("state[%d]=%s", i, states[i])
		}
	}
	s.Reset()
	if got := s.Observe("BTC-USD", 1).State; got != SequenceFirst {
		t.Fatalf("reset=%s", got)
	}
}
