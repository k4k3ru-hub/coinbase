package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	intxmarketdata "github.com/k4k3ru-hub/coinbase/go/intx/marketdata"
	"github.com/k4k3ru-hub/coinbase/go/marketdata"
)

type fakeServerTimeGetter struct {
	result *marketdata.Time
	err    error
}

func (f fakeServerTimeGetter) GetServerTime(context.Context) (*marketdata.Time, error) {
	return f.result, f.err
}

type fakePerpetualInstrumentLister struct {
	result []intxmarketdata.Instrument
	err    error
}

func (f fakePerpetualInstrumentLister) ListPerpetualInstruments(context.Context) ([]intxmarketdata.Instrument, error) {
	return f.result, f.err
}

func TestRESTTimeCommand(t *testing.T) {
	t.Parallel()
	commandLine, err := newCLI(fakeServerTimeGetter{result: &marketdata.Time{
		ISO:   "2026-08-19T00:00:00.123456Z",
		Epoch: json.RawMessage("1755561600.123456"),
	}}, fakePerpetualInstrumentLister{})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	var errorOutput bytes.Buffer
	if err := commandLine.SetIO(strings.NewReader(""), &output, &errorOutput); err != nil {
		t.Fatal(err)
	}
	if err := commandLine.RunArgs([]string{"rest", "time"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"iso": "2026-08-19T00:00:00.123456Z"`) {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func TestRESTTimeCommandError(t *testing.T) {
	t.Parallel()
	commandLine, err := newCLI(fakeServerTimeGetter{err: errors.New("unavailable")}, fakePerpetualInstrumentLister{})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	var errorOutput bytes.Buffer
	if err := commandLine.SetIO(strings.NewReader(""), &output, &errorOutput); err != nil {
		t.Fatal(err)
	}
	err = commandLine.RunArgs([]string{"rest", "time"})
	if err == nil || !strings.Contains(err.Error(), "failed to run coinbase exchange rest time command") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewCLINilDependency(t *testing.T) {
	t.Parallel()
	if _, err := newCLI(nil, fakePerpetualInstrumentLister{}); err == nil || !strings.Contains(err.Error(), "time_getter=null") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := newCLI(fakeServerTimeGetter{}, nil); err == nil || !strings.Contains(err.Error(), "instrument_lister=null") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestINTXInstrumentsCommand(t *testing.T) {
	t.Parallel()
	commandLine, err := newCLI(fakeServerTimeGetter{}, fakePerpetualInstrumentLister{result: []intxmarketdata.Instrument{{
		Symbol:         "BTC-PERP",
		Type:           intxmarketdata.InstrumentTypePerpetual,
		BaseAssetName:  "BTC",
		QuoteAssetName: "USDC",
		BaseIncrement:  "0.0001",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	var errorOutput bytes.Buffer
	if err := commandLine.SetIO(strings.NewReader(""), &output, &errorOutput); err != nil {
		t.Fatal(err)
	}
	if err := commandLine.RunArgs([]string{"intx", "instruments"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"symbol": "BTC-PERP"`) || !strings.Contains(output.String(), `"type": "PERP"`) {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func TestINTXInstrumentsCommandError(t *testing.T) {
	t.Parallel()
	commandLine, err := newCLI(fakeServerTimeGetter{}, fakePerpetualInstrumentLister{err: errors.New("unavailable")})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	var errorOutput bytes.Buffer
	if err := commandLine.SetIO(strings.NewReader(""), &output, &errorOutput); err != nil {
		t.Fatal(err)
	}
	err = commandLine.RunArgs([]string{"intx", "instruments"})
	if err == nil || !strings.Contains(err.Error(), "failed to run coinbase intx instruments command") {
		t.Fatalf("unexpected error: %v", err)
	}
}
