package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/coinbase/go/marketdata"
)

type fakeServerTimeGetter struct {
	result *marketdata.Time
	err    error
}

func (f fakeServerTimeGetter) GetServerTime(context.Context) (*marketdata.Time, error) {
	return f.result, f.err
}

func TestRESTTimeCommand(t *testing.T) {
	t.Parallel()
	commandLine, err := newCLI(fakeServerTimeGetter{result: &marketdata.Time{
		ISO:   "2026-08-19T00:00:00.123456Z",
		Epoch: json.RawMessage("1755561600.123456"),
	}})
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
	commandLine, err := newCLI(fakeServerTimeGetter{err: errors.New("unavailable")})
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
	if _, err := newCLI(nil); err == nil || !strings.Contains(err.Error(), "time_getter=null") {
		t.Fatalf("unexpected error: %v", err)
	}
}
