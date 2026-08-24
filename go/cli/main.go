package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/k4k3ru-hub/cli/go"
	intxmarketdata "github.com/k4k3ru-hub/coinbase/go/intx/marketdata"
	intxrest "github.com/k4k3ru-hub/coinbase/go/intx/rest"
	"github.com/k4k3ru-hub/coinbase/go/marketdata"
	"github.com/k4k3ru-hub/coinbase/go/rest"
)

const version = "1.1.0"

type serverTimeGetter interface {
	GetServerTime(context.Context) (*marketdata.Time, error)
}

type perpetualInstrumentLister interface {
	ListPerpetualInstruments(context.Context) ([]intxmarketdata.Instrument, error)
}

func main() {
	restClient, err := rest.NewClient(nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	intxRESTClient, err := intxrest.NewClient(nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	commandLine, err := newCLI(restClient.MarketData(), intxRESTClient.MarketData())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := commandLine.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCLI(timeGetter serverTimeGetter, instrumentLister perpetualInstrumentLister) (*cli.CLI, error) {
	if timeGetter == nil {
		return nil, fmt.Errorf("failed to create coinbase exchange cli: time_getter=null")
	}
	if instrumentLister == nil {
		return nil, fmt.Errorf("failed to create coinbase exchange cli: instrument_lister=null")
	}

	commandLine := cli.NewCLI(nil)
	commandLine.SetVersion(version)
	if err := commandLine.Root().AddDefaultConfigOption(); err != nil {
		return nil, fmt.Errorf("failed to create coinbase exchange cli: %w", err)
	}

	restCommand := cli.NewCommand("rest")
	restCommand.SetUsage("Coinbase Exchange public REST API commands.")
	if err := commandLine.Root().AddCommand(restCommand); err != nil {
		return nil, fmt.Errorf("failed to create coinbase exchange cli: %w", err)
	}

	timeCommand := cli.NewCommand("time")
	timeCommand.SetUsage("Get Coinbase Exchange server time.")
	timeCommand.SetAction(func(ctx *cli.Context) error {
		result, err := timeGetter.GetServerTime(context.Background())
		if err != nil {
			return fmt.Errorf("failed to run coinbase exchange rest time command: %w", err)
		}
		encoder := json.NewEncoder(ctx.Output())
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("failed to run coinbase exchange rest time command: failed to output result: %w", err)
		}
		return nil
	})
	if err := restCommand.AddCommand(timeCommand); err != nil {
		return nil, fmt.Errorf("failed to create coinbase exchange cli: %w", err)
	}

	intxCommand := cli.NewCommand("intx")
	intxCommand.SetUsage("Coinbase International Exchange perpetual market data commands.")
	if err := commandLine.Root().AddCommand(intxCommand); err != nil {
		return nil, fmt.Errorf("failed to create coinbase exchange cli: %w", err)
	}

	instrumentsCommand := cli.NewCommand("instruments")
	instrumentsCommand.SetUsage("List Coinbase International Exchange perpetual instruments.")
	instrumentsCommand.SetAction(func(ctx *cli.Context) error {
		result, err := instrumentLister.ListPerpetualInstruments(context.Background())
		if err != nil {
			return fmt.Errorf("failed to run coinbase intx instruments command: %w", err)
		}
		encoder := json.NewEncoder(ctx.Output())
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("failed to run coinbase intx instruments command: failed to output result: %w", err)
		}
		return nil
	})
	if err := intxCommand.AddCommand(instrumentsCommand); err != nil {
		return nil, fmt.Errorf("failed to create coinbase exchange cli: %w", err)
	}

	return commandLine, nil
}
